package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/ent/schema/mixin"
)

// EventLog holds the schema definition for the EventLog entity.
// Lobby Event(GameEventManager) 상태변경을 서버에 적재하는 관측용 로그.
// 진실 원천은 여전히 클라 로컬(PlayerPrefs) — 이 로그로 게임 상태를 강제/롤백하지 않는다.
type EventLog struct {
	ent.Schema
}

// Mixin of the EventLog. (id / created_at / updated_at 자동)
func (EventLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.GlobalMixin{},
	}
}

// Fields of the EventLog.
func (EventLog) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("user_id", uuid.UUID{}).
			Comment("유저 ID (세션에서 도출, body 에 실리지 않음)"),
		field.Text("event_key").
			Comment("GameEvent enum 이름 (문자열, 예 \"FirstPlay\"). 서버 화이트리스트 검증 없음."),
		field.Text("state_before").
			Comment("변경 전 GameEventState 이름 (문자열, 예 \"Locked\")"),
		field.Text("state_after").
			Comment("변경 후 GameEventState 이름 (문자열, 예 \"Available\")"),
		field.Time("changed_at").
			Comment("클라 발생 시각 (클라가 unix seconds 로 전송 → time 변환). created_at 은 서버 수신 시각."),
		field.String("client_log_id").Optional().Nillable().Unique().MaxLen(36).
			Comment("클라이언트 발급 멱등성 키 (UUID v4). 단건/batch 공통."),
	}
}

// Edges of the EventLog.
func (EventLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("event_logs").
			Field("user_id").
			Unique().
			Required(),
	}
}

// Indexes of the EventLog.
func (EventLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("user_id", "event_key"),
	}
}
