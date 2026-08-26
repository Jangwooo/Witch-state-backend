package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/ent/schema/mixin"
)

// Record holds the schema definition for the Record entity.
type Record struct {
	ent.Schema
}

// Mixin of the Record.
func (Record) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.GlobalMixin{},
	}
}

// Fields of the Record.
func (Record) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("user_id", uuid.UUID{}).
			Comment("유저 ID"),
		field.Text("music_id").
			Comment("음악 ID"),
		field.Text("stage_id").
			Comment("스테이지 ID"),
		field.Int("score").
			Comment("점수"),
		field.Int("perfect_count").Default(0).
			Comment("Perfect 개수"),
		field.Int("good_count").Default(0).
			Comment("Good 개수"),
		field.Int("bad_count").Default(0).
			Comment("Bad 개수"),
		field.Int("miss_count").Default(0).
			Comment("Miss 개수"),
		field.Int("max_combo").Default(0).
			Comment("최대 콤보"),
		field.Float("accuracy").Default(0).
			Comment("정확도 (%)"),
		field.Enum("rank").Values("F", "D", "D_P", "C", "C_P", "B", "B_P", "A", "A_P", "S", "SS", "SSS").Optional().
			Comment("랭크. _P 접미사는 Plus 등급(예: A_P = A+). S 이상은 Plus 없음."),
		field.Bool("is_full_combo").Default(false).
			Comment("풀콤보 여부"),
		field.Bool("is_perfect_play").Default(false).
			Comment("퍼펙트 플레이 여부"),
		field.Enum("game_status").
			Values("completed", "gave_up", "retry", "failed").
			Default("completed").
			Comment("게임 플레이 상태 (completed: 완료, gave_up: 포기, retry: 재도전, failed: 실패)"),
		field.JSON("additional_info", map[string]interface{}{}).Optional().
			Comment("추가 정보"),
		field.Bool("is_valid").Default(true).
			Comment("유효한 기록 여부"),
		field.String("client_record_id").Optional().Nillable().Unique().MaxLen(36).
			Comment("클라이언트 발급 멱등성 키 (batch 송신 시. 단건 POST 에선 null)"),
	}
}

// Edges of the Record.
func (Record) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("records").
			Field("user_id").
			Unique().
			Required(),
		edge.From("music", Music.Type).
			Ref("records").
			Field("music_id").
			Unique().
			Required(),
		edge.From("stage", Stage.Type).
			Ref("records").
			Field("stage_id").
			Unique().
			Required(),
	}
}

// Indexes of the Record.
func (Record) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "music_id", "stage_id"),
		index.Fields("user_id"),
		index.Fields("music_id"),
		index.Fields("stage_id"),
	}
}
