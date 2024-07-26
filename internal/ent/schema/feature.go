package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Feature struct {
	ent.Schema
}

func (Feature) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.TimeMixin{},
		entutils.MetadataAnnotationsMixin{},
	}
}

func (Feature) Fields() []ent.Field {
	return []ent.Field{
		field.String("namespace").NotEmpty().Immutable(),
		field.String("name").NotEmpty(),
		field.String("key").NotEmpty().Immutable(),
		field.String("meter_slug").Optional().Nillable().Immutable(),
		field.JSON("meter_group_by_filters", map[string]string{}).Optional(),
		field.Time("archived_at").Optional().Nillable(),
	}
}

func (Feature) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
	}
}

func (Feature) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("entitlement", Entitlement.Type),
	}
}
