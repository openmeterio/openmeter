package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Feature struct {
	ent.Schema
}

// Mixin of the Feature.
func (Feature) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.TimeMixin{},
	}
}

// Fields of the Feature.
func (Feature) Fields() []ent.Field {
	return []ent.Field{
		field.String("namespace").NotEmpty().Immutable(),
		field.String("name").NotEmpty(),
		field.String("meter_slug").NotEmpty().Immutable(),
		field.JSON("meter_group_by_filters", map[string]string{}).Optional(),
		field.Time("archived_at").Optional().Nillable(),
	}
}

// Indexes of the Feature.
func (Feature) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
	}
}

// TODO: link to entitlements
// Edges of the Feature define the relations to other entities.
// func (Feature) Edges() []ent.Edge {
// 	return []ent.Edge{
// 		edge.
// 			To("credit_grants", CreditEntry.Type).
// 			Annotations(entsql.OnDelete(entsql.Restrict)),
// 	}
// }
