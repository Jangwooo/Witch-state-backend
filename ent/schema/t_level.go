package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// TLevel 레벨 정의 테이블 (t_level)
type TLevel struct {
	ent.Schema
}

func (TLevel) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "t_level"},
	}
}

func (TLevel) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			Unique().
			Immutable().
			Comment("레벨"),
		field.Int("require_exp").
			Comment("요구 경험치"),
	}
}
