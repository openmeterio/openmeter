package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Feature struct {
	ent.Schema
}

// Mixin of the Feature.
func (Feature) Mixin() []ent.Mixin {
	return []ent.Mixin{
		IDMixin{},
		TimeMixin{},
	}
}

// Fields of the Feature.
func (Feature) Fields() []ent.Field {
	return []ent.Field{
		field.String("namespace").NotEmpty().Immutable(),
		field.String("name").NotEmpty(),
		field.String("meter_slug").NotEmpty().Immutable(),
		field.JSON("meter_group_by_filters", map[string]string{}).Optional(),
		field.Bool("archived").Default(false),
	}
}

// Indexes of the Feature.
func (Feature) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
	}
}

// Edges of the Feature define the relations to other entities.
func (Feature) Edges() []ent.Edge {
	return []ent.Edge{
		edge.
			To("credit_grants", CreditEntry.Type).
			Annotations(entsql.OnDelete(entsql.Restrict)),
	}
}
