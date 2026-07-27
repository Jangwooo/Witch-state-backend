#!/usr/bin/env bash
# =============================================================================
# 전시(EXHIBITION) record 적재 진단 스크립트  (exhibition-logging, 2026-07-28)
# -----------------------------------------------------------------------------
# 증상: 실제 Exhibition 빌드 플레이를 했는데 "records 테이블에 기록이 없다".
#
# ⚠️ 핵심 함정: 클라 JsonUtility 버그 수정으로 additional_info 마커가 이제 실제로
#    도달하기 시작했다. 마커가 도달하면 서버는 설계대로 is_valid=false 로 저장한다
#    (repository/record.go:63). 그리고 List/Best API 는 is_valid=true 만 조회한다
#    (record.go:106,126). 즉:
#
#      "게임/뷰어에서 안 보인다"  ≠  "DB에 없다"
#
#    Q2 가 처음으로 정상 발동하면 조회 API 에서 사라지는 것이 정상 동작이다.
#    따라서 반드시 is_valid 필터 없이 raw SELECT 로 확인해야 한다.
#
# 사용법: 운영 서버 디렉토리(.env.prod 위치)에서
#     bash check-exhibition-record.sh            # 최근 24시간
#     HOURS=72 bash check-exhibition-record.sh   # 범위 변경
#   (읽기 전용 — SELECT 만 수행하며 아무것도 변경하지 않는다.)
# =============================================================================
set -uo pipefail

ENV_FILE="${ENV_FILE:-.env.prod}"
HOURS="${HOURS:-24}"
PASS="✅"; FAIL="❌"; WARN="⚠️"; INFO="ℹ️"

echo "=============================================================="
echo " 전시 record 적재 진단  ($(date -u +%Y-%m-%dT%H:%M:%SZ), 최근 ${HOURS}h)"
echo "=============================================================="

if [ ! -f "$ENV_FILE" ]; then
  echo "$FAIL [$ENV_FILE] 없음. 운영 디렉토리에서 실행하거나 ENV_FILE=경로 로 지정."
  exit 1
fi
set -a; . "./$ENV_FILE" >/dev/null 2>&1; set +a

uid="${EXHIBITION_USER_ID:-}"
if [ -z "$uid" ]; then
  echo "$FAIL EXHIBITION_USER_ID 미설정 → 게이트 비활성. check-exhibition-gate.sh 를 먼저 실행하라."
  exit 1
fi
echo "$PASS 전시 고정계정: $uid"

run_psql() {
  if command -v psql >/dev/null 2>&1 && [ -n "${DB_HOST:-}" ]; then
    PGPASSWORD="${DB_PASSWORD:-}" psql -h "${DB_HOST}" -p "${DB_PORT:-5432}" \
      -U "${DB_USER}" -d "${DB_NAME}" -tAF'|' -c "$1" 2>&1
  elif command -v docker >/dev/null 2>&1; then
    local c="${PG_CONTAINER:-}"
    if [ -z "$c" ]; then
      c=$(docker ps --format '{{.Names}}' | grep -iE 'postgres|db|pg' | head -1)
    fi
    if [ -n "$c" ]; then
      docker exec -e PGPASSWORD="${DB_PASSWORD:-}" "$c" \
        psql -U "${DB_USER}" -d "${DB_NAME}" -tAF'|' -c "$1" 2>&1
    else
      echo "__NO_PSQL__"
    fi
  else
    echo "__NO_PSQL__"
  fi
}

# --- 1) is_valid 필터 없는 raw 조회 (핵심) ----------------------------------
Q1="SELECT id, created_at, score, is_valid, game_status,
           COALESCE(additional_info::text,'(null)')
    FROM records
    WHERE user_id = '${uid}'
      AND created_at > NOW() - INTERVAL '${HOURS} hours'
    ORDER BY created_at DESC LIMIT 30;"

echo
echo "-- [1] 전시 계정 최근 record (is_valid 필터 없음) --"
out=$(run_psql "$Q1")

if [ "$out" = "__NO_PSQL__" ]; then
  echo "$WARN psql/DB 접근 경로를 못 찾음. 아래 쿼리를 DB 콘솔에서 직접 실행하라:"
  echo "$Q1"
  exit 0
elif echo "$out" | grep -qiE '^psql:|error|does not exist|FATAL'; then
  echo "$WARN DB 조회 오류:"; echo "$out"; exit 1
elif [ -z "$out" ]; then
  echo "$FAIL 0행 — 전시 계정에 record 가 실제로 없다. 요청이 서버에 도달하지 못한 것."
  echo "    → 원인 후보: (a) 게이트 비활성/401, (b) 클라 미배포(구 빌드), (c) 네트워크,"
  echo "       (d) 400 sanity reject. app 컨테이너 로그에서 POST /api/v1/records 확인:"
  echo "       docker logs --since ${HOURS}h \$(docker ps --format '{{.Names}}' | grep -iE 'app|backend' | head -1) 2>&1 | grep -i record"
else
  echo "$PASS record 존재 (컬럼: id|created_at|score|is_valid|game_status|additional_info)"
  echo "$out" | while IFS= read -r line; do echo "    $line"; done
  echo
  n_total=$(echo "$out" | grep -c .)
  n_exh=$(echo "$out" | grep -c 'is_exhibition')
  n_invalid=$(echo "$out" | awk -F'|' '$4=="f"' | grep -c .)
  echo "$INFO 총 ${n_total}행 / 전시마커 있음 ${n_exh}행 / is_valid=false ${n_invalid}행"
  echo
  if [ "$n_exh" -gt 0 ] && [ "$n_invalid" -gt 0 ]; then
    echo "$PASS 🎯 Q2 가드 최초 발동 확인 — 마커 도달 + is_valid=false 저장."
    echo "    조회 API(List/Best)에서 안 보이는 것은 설계대로의 정상 동작이다."
    echo "    Q5 표시명은 위 additional_info.exhibition_name 값으로 확인하라."
  elif [ "$n_exh" -eq 0 ]; then
    echo "$WARN record 는 있으나 is_exhibition 마커가 없다 → 클라 수정본 미배포(구 빌드) 의심."
    echo "    is_valid=true 로 일반 기록 적재 + EXP 부여됐을 것 (2026-07-14 와 동일 증상)."
  fi
fi

# --- 2) EXP 미부여(Q3) 확인 --------------------------------------------------
echo
echo "-- [2] 전시 계정 exp/level (Q3: 절대 증가하면 안 됨) --"
out2=$(run_psql "SELECT level, exp FROM users WHERE id = '${uid}';")
[ "$out2" != "__NO_PSQL__" ] && echo "    level|exp = $out2"

echo "=============================================================="
echo "$INFO 소급 정정은 lead 결정으로 우선순위 하향됨(전시는 EXP 미사용, 클라 이중방어)."
echo "   본 스크립트는 읽기 전용이며 어떤 데이터도 변경하지 않았다."
echo "=============================================================="
