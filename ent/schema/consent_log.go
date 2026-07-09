package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/ent/schema/mixin"
)

// ConsentLog holds the schema definition for the ConsentLog entity.
// EULA/개인정보 동의를 서버에 적재하는 법적 증빙 로그 (consent-log 아젠다).
//
// event/record 로그와 결정적으로 다른 점:
//   - 로그인 이전에 발생 → user_id nullable, 식별 주체는 client_id.
//   - 모드 무관·무조건 적재 (게이팅 없음).
//   - 무인증 라우트 (auth/ban/hmac 미적용).
//   - 법적 보존: policy_version 필수, 파기는 expires_at 로 관리(탈퇴 후 3년).
type ConsentLog struct {
	ent.Schema
}

// Mixin of the ConsentLog. (id / created_at / updated_at 자동)
func (ConsentLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.GlobalMixin{},
	}
}

// Fields of the ConsentLog.
//
// ⚠️ 이 정의는 원격 Postgres 에 이미 손수 SQL 로 생성된 consent_logs 테이블과 100% 정합해야 한다
// (server-side 프롬프트 §0). 컬럼 타입·nullable·UNIQUE 가 어긋나면 부팅 auto-migrate 가 충돌한다.
//   - client_consent_id / client_id / user_id 는 uuid 타입.
//   - user_id 는 FK 없음(인덱스만) — 로그인 전 NULL, signin 에서 나중 UPDATE.
//   - received_at 은 GlobalMixin 의 created_at 과 별개인 실제 컬럼(서버 수신 시각).
func (ConsentLog) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("client_consent_id", uuid.UUID{}).Unique().
			Comment("클라이언트 발급 멱등성 키 (UUID v4). 단건/batch 공통."),
		field.UUID("client_id", uuid.UUID{}).
			Comment("기기 고유 UUID (로그인 무관 지속). 식별 주체."),
		// user_id 는 nullable + FK 없음 — 동의는 로그인 전 발생. signin 시 자동 연결로 나중에 UPDATE.
		field.UUID("user_id", uuid.UUID{}).Optional().Nillable().
			Comment("로그인 사용자 ID. 동의 시점엔 보통 null, signin 자동연결로 나중에 채워짐. FK 없음."),
		field.Enum("consent_type").Values("eula", "privacy").
			Comment("동의 항목 (eula / privacy). 항목별 개별 레코드."),
		field.String("policy_version").NotEmpty().MaxLen(64).
			Comment("동의한 정책 문서 버전 문자열 (예 \"eula@2026-07-10\"). 법적으로 결정적. 화이트리스트 검증 없음."),
		field.Bool("granted").
			Comment("동의=true / 거부=false."),
		field.String("nickname").Optional().MaxLen(64).
			Comment("입력 표시명 (익명이면 빈 문자열)."),
		field.Time("consented_at").
			Comment("동의 시각 (클라 발생, unix seconds 수신 → time 변환)."),
		field.String("client_version").Optional().MaxLen(32).
			Comment("클라 버전 (증빙 맥락)."),
		field.String("platform").Optional().MaxLen(32).
			Comment("실행 플랫폼."),
		field.String("locale").Optional().MaxLen(16).
			Comment("동의 화면 언어."),
		field.String("build_mode").Optional().MaxLen(32).
			Comment("빌드 모드 (Inhouse/Playtest/Demo/Public/Exhibition/OnlineExhibition)."),
		field.Time("received_at").Default(time.Now).
			Comment("서버 수신 시각 (default now()). GlobalMixin created_at 과 별개 실제 컬럼."),
		field.Time("expires_at").Optional().Nillable().
			Comment("파기 예정 시각. 회원탈퇴 시 탈퇴시점+3년으로 설정. 미탈퇴 계정은 null(파기 안 함)."),
	}
}

// Edges of the ConsentLog.
// ⚠️ user_id 에 FK(users edge)를 걸지 않는다 (server-side §0-3): consent 는 로그인 전 NULL 로 오고
// signin 에서 나중에 UPDATE 되는 흐름이라 FK 를 안 건다고 확정. user_id 는 인덱스로만 처리.
func (ConsentLog) Edges() []ent.Edge {
	return nil
}

// Indexes of the ConsentLog.
// ⚠️ 인덱스 이름을 실제 DB(§0)에 맞춘다. 특히 복합 인덱스는 ent 기본 생성명
// (consentlog_consent_type_policy_version)이 실제 DB 이름(consentlog_ctype_pversion)과 달라
// StorageKey 로 고정하지 않으면 auto-migrate 가 같은 컬럼에 중복 인덱스를 만들 수 있다.
func (ConsentLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("client_id").StorageKey("consentlog_client_id"),
		index.Fields("user_id").StorageKey("consentlog_user_id"),
		index.Fields("consent_type", "policy_version").StorageKey("consentlog_ctype_pversion"),
		index.Fields("expires_at").StorageKey("consentlog_expires_at"),
	}
}
