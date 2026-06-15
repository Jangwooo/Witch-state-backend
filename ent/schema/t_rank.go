package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// TRank 랭크별 경험치 계수 테이블 (t_rank)
type TRank struct {
	ent.Schema
}

func (TRank) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "t_rank"},
	}
}

func (TRank) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			StorageKey("Rank").
			Immutable().
			Comment("랭크"),
		field.Float32("coefficient").
			Comment("경험치 계수"),
	}
}
