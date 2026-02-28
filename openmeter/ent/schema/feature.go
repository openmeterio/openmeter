package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Feature struct {
	ent.Schema
}

func (Feature) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.TimeMixin{},
		entutils.MetadataMixin{},
	}
}

func (Feature) Fields() []ent.Field {
	return []ent.Field{
		field.String("namespace").NotEmpty().Immutable(),
		field.String("name").NotEmpty(),
		field.String("key").NotEmpty().Immutable(),
		field.String("meter_slug").Optional().Nillable().Immutable(),
		field.JSON("meter_group_by_filters", map[string]string{}).Optional(),
		field.JSON("advanced_meter_group_by_filters", feature.MeterGroupByFilters{}).Optional(),
		field.String("unit_cost_type").Optional().Nillable(),
		field.Other("unit_cost_manual_amount", alpacadecimal.Decimal{}).
			Optional().Nillable().
			SchemaType(map[string]string{dialect.Postgres: "numeric"}),
		field.String("unit_cost_llm_provider_property").Optional().Nillable(),
		field.String("unit_cost_llm_provider").Optional().Nillable(),
		field.String("unit_cost_llm_model_property").Optional().Nillable(),
		field.String("unit_cost_llm_model").Optional().Nillable(),
		field.String("unit_cost_llm_token_type_property").Optional().Nillable(),
		field.String("unit_cost_llm_token_type").Optional().Nillable(),
		field.Time("archived_at").Optional().Nillable(),
	}
}

func (Feature) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "key").
			Annotations(
				entsql.IndexWhere("archived_at IS NULL"),
			).
			Unique(),
		index.Fields("namespace", "meter_slug").
			Annotations(
				entsql.IndexWhere("archived_at IS NULL"),
			),
	}
}

func (Feature) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("entitlement", Entitlement.Type),
		edge.To("ratecard", PlanRateCard.Type),
		edge.To("addon_ratecard", AddonRateCard.Type),
		// FIXME: enable foreign key constraints
		// edge.From("meter", Meter.Type).
		// 	Ref("feature").
		// 	Field("meter_slug").
		// 	Required().
		// 	Immutable(),
	}
}
