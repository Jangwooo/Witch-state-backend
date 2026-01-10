package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Stage holds the schema definition for the Stage entity.
type Stage struct {
	ent.Schema
}

// Fields of the Stage.
func (Stage) Fields() []ent.Field {
	return []ent.Field{
		field.Text("id").
			Immutable().
			Unique().
			Comment("곡 ID"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("Created time"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("Updated time"),

		field.Text("music_id").
			Comment("음악 ID"),
		field.Text("level_name").NotEmpty().
			Comment("난이도 이름 (Easy, Normal, Hard, Expert)"),
		field.Int("difficulty").
			Comment("난이도 수치 (1-10)"),
		field.Int("total_notes").
			Comment("총 노트 수"),
		field.Int("max_combo").
			Comment("최대 콤보"),
		field.Bool("is_active").Default(true).
			Comment("활성 여부"),
	}
}

// Edges of the Stage.
func (Stage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("music", Music.Type).
			Ref("stages").
			Field("music_id").
			Required().
			Unique(),
		edge.To("records", Record.Type),
	}
}
