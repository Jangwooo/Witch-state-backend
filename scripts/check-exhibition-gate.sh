#!/usr/bin/env bash
# =============================================================================
# 전시(EXHIBITION) 게이트 진단 스크립트  (exhibition-logging 채널, 2026-07-13)
# -----------------------------------------------------------------------------
# 증상: 전시 record 401. 원인 후보 = (B) 운영 env 미설정 or 전시 전용 계정 DB 미생성.
#
# 게이트 활성 조건(코드 확정, middleware/exhibition.go:40):
#     EXHIBITION_KEY != ""  AND  EXHIBITION_USER_ID 계정이 DB에 실존
#   → 둘 다여야 활성. 하나라도 비면 게이트 죽고 전시 record 는 일반 auth 폴백 → 401.
#
# 사용법: docker-compose.prod.yml + .env.prod 가 있는 운영 서버 디렉토리에서
#     bash check-exhibition-gate.sh
#   (읽기 전용 — env/DB 를 조회만 하고 아무것도 변경하지 않는다.)
# =============================================================================
set -uo pipefail

ENV_FILE="${ENV_FILE:-.env.prod}"
PASS="✅"; FAIL="❌"; WARN="⚠️"
fail=0

echo "=============================================================="
echo " 전시 게이트 진단  ($(date -u +%Y-%m-%dT%H:%M:%SZ))"
echo "=============================================================="

# --- 0) env 파일 존재 -------------------------------------------------------
if [ ! -f "$ENV_FILE" ]; then
  echo "$FAIL [$ENV_FILE] 없음. 운영 디렉토리에서 실행하거나 ENV_FILE=경로 로 지정."
  exit 1
fi
echo "$PASS env 파일: $ENV_FILE"

# .env.prod 로드 (값은 마스킹해서만 출력)
set -a; . "./$ENV_FILE" >/dev/null 2>&1; set +a

# --- 1) EXHIBITION_KEY 존재 -------------------------------------------------
if [ -n "${EXHIBITION_KEY:-}" ]; then
  klen=${#EXHIBITION_KEY}
  echo "$PASS EXHIBITION_KEY 설정됨 (길이 ${klen}, 앞 8자: ${EXHIBITION_KEY:0:8}…)"
else
  echo "$FAIL EXHIBITION_KEY 미설정 → 게이트 비활성 → 전시 record 401 원인 확정."
  fail=1
fi

# --- 2) EXHIBITION_USER_ID 존재 + UUID 형식 ---------------------------------
uid="${EXHIBITION_USER_ID:-}"
if [ -n "$uid" ]; then
  if echo "$uid" | grep -qiE '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'; then
    echo "$PASS EXHIBITION_USER_ID 설정됨 (UUID 형식 OK): $uid"
  else
    echo "$FAIL EXHIBITION_USER_ID 값이 UUID 형식이 아님: '$uid' → uuid.Parse 실패 → 게이트 비활성."
    fail=1
  fi
else
  echo "$FAIL EXHIBITION_USER_ID 미설정 → 게이트 비활성 → 전시 record 401 원인 확정."
  fail=1
fi

# --- 3) 전시 전용 계정이 DB(users)에 실존하는가 -----------------------------
# ⚠️ 가장 흔한 누락 포인트: 키만 넣고 §2 계정 수동생성을 안 한 경우.
# ⚠️ 스키마 주의: users.platform_type enum 은 ('steam','stove') 뿐 — 'exhibition' 불가.
#    전시 계정은 steam/stove 중 하나로 생성돼 있어야 한다 (OPS.md §2 의 예시 'exhibition' 은 오기).
if [ -n "$uid" ] && echo "$uid" | grep -qiE '^[0-9a-f-]{36}$'; then
  echo "-- users 테이블 조회 시도 (읽기 전용 SELECT) --"
  Q="SELECT id, platform_type, platform_user_id, is_banned, level, exp FROM users WHERE id = '${uid}';"

  run_psql() {
    # 우선순위: (a) 로컬 psql, (b) app 컨테이너에서 psql, (c) postgres 컨테이너에서 psql
    if command -v psql >/dev/null 2>&1 && [ -n "${DB_HOST:-}" ]; then
      PGPASSWORD="${DB_PASSWORD:-}" psql -h "${DB_HOST}" -p "${DB_PORT:-5432}" \
        -U "${DB_USER}" -d "${DB_NAME}" -tAc "$1" 2>&1
    elif command -v docker >/dev/null 2>&1; then
      # postgres 컨테이너명이 다르면 PG_CONTAINER=이름 로 지정
      local c="${PG_CONTAINER:-}"
      if [ -z "$c" ]; then
        c=$(docker ps --format '{{.Names}}' | grep -iE 'postgres|db|pg' | head -1)
      fi
      if [ -n "$c" ]; then
        docker exec -e PGPASSWORD="${DB_PASSWORD:-}" "$c" \
          psql -U "${DB_USER}" -d "${DB_NAME}" -tAc "$1" 2>&1
      else
        echo "__NO_PSQL__"
      fi
    else
      echo "__NO_PSQL__"
    fi
  }

  out=$(run_psql "$Q")
  if [ "$out" = "__NO_PSQL__" ]; then
    echo "$WARN psql/DB 접근 경로를 못 찾음. 아래 쿼리를 DB 콘솔에서 직접 실행해 1행 나오는지 확인:"
    echo "    $Q"
    echo "    (0행이면 → 전시 계정 미생성 = 401 원인. 1행이면 계정 OK.)"
  elif [ -z "$out" ]; then
    echo "$FAIL users 에 id=$uid 계정 없음 (0행) → 게이트 비활성 → 401 원인 확정."
    echo "    → EXHIBITION_OPS.md §2 대로 전시 전용 계정 1개를 수동 생성해야 함 (platform_type=steam|stove)."
    fail=1
  elif echo "$out" | grep -qiE 'error|does not exist|FATAL'; then
    echo "$WARN DB 조회 오류(접속/권한 문제일 수 있음). 원문:"; echo "    $out"
  else
    echo "$PASS 전시 계정 실존:"
    echo "    $out"
    echo "    (컬럼: id | platform_type | platform_user_id | is_banned | level | exp)"
    if echo "$out" | awk -F'|' '{exit ($4=="t")?0:1}'; then
      echo "$WARN 이 계정 is_banned=true → 밴 미들웨어에서 막힐 수 있음. 밴 해제 필요."
      fail=1
    fi
  fi
fi

# --- 4) 컨테이너 배포 상태 (이미지가 819ae5a 반영본인지 간접 확인) -----------
if command -v docker >/dev/null 2>&1; then
  echo "-- docker 컨테이너 상태 --"
  docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}' 2>/dev/null | grep -iE 'NAMES|witch|lounge|backend|app|redis|postgres|db' || echo "$WARN 관련 컨테이너를 못 찾음."
  echo "$WARN env 변경 후엔 반드시 재기동(핫리로드 없음):"
  echo "    docker compose -f docker-compose.prod.yml up -d"
fi

# --- 결론 -------------------------------------------------------------------
echo "=============================================================="
if [ "$fail" -eq 0 ]; then
  echo "$PASS 게이트 활성 조건 충족으로 보임 (KEY+USER_ID+계정 실존)."
  echo "   그래도 401 이면: (a) 컨테이너가 최신 이미지로 재기동 안 됐거나,"
  echo "   (b) 클라 부착 키 값과 서버 EXHIBITION_KEY 값이 불일치. 두 값을 대조하라."
else
  echo "$FAIL 위 $FAIL 항목이 401 원인. 해당 항목 조치 후 컨테이너 재기동하면 해결."
fi
echo "=============================================================="
