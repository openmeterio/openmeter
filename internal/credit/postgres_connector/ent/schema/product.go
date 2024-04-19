package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Product struct {
	ent.Schema
}

// Mixin of the Product.
func (Product) Mixin() []ent.Mixin {
	return []ent.Mixin{
		IDMixin{},
		TimeMixin{},
	}
}

// Fields of the Product.
func (Product) Fields() []ent.Field {
	return []ent.Field{
		field.String("namespace").NotEmpty().Immutable(),
		field.String("name").NotEmpty(),
		field.String("meter_slug").NotEmpty().Immutable(),
		field.JSON("meter_group_by_filters", map[string]string{}).Optional(),
		field.Bool("archived").Default(false),
	}
}

// Indexes of the Product.
func (Product) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
	}
}

// Edges of the Product define the relations to other entities.
func (Product) Edges() []ent.Edge {
	return []ent.Edge{
		edge.
			To("credit_grants", CreditEntry.Type).
			Annotations(entsql.OnDelete(entsql.Restrict)),
	}
}
