package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Music holds the schema definition for the Music entity.
type Music struct {
	ent.Schema
}

// Fields of the Music.
func (Music) Fields() []ent.Field {
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

		field.Text("name").
			Comment("곡 제목"),
		field.Text("artist").
			Comment("아티스트"),
		field.Text("composer").Optional().
			Comment("작곡가"),
		field.Float("bpm").
			Comment("BPM"),
		field.Text("genre").Optional().
			Comment("장르"),
		field.Text("description").Optional().
			Comment("곡 설명"),
		field.Bool("is_recommended").Default(false).
			Comment("추천곡 여부"),
		field.Bool("is_free").Default(true).
			Comment("무료곡 여부"),
		field.Int("unlock_level").Default(1).
			Comment("해금 레벨"),
		field.Time("release_date").Optional().Nillable().
			Comment("출시일"),
		field.Bool("is_active").Default(true).
			Comment("활성 여부"),
	}
}

// Edges of the Music.
func (Music) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("stages", Stage.Type),
		edge.To("records", Record.Type),
	}
}
