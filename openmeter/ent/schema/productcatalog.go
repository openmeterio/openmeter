package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Plan struct {
	ent.Schema
}

func (Plan) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.UniqueResourceMixin{},
	}
}

func (Plan) Fields() []ent.Field {
	return []ent.Field{
		field.Int("version").
			Min(1),
		field.String("currency").
			Default("USD").
			NotEmpty().
			Immutable(),
		field.Time("effective_from").
			Optional().
			Nillable(),
		field.Time("effective_to").
			Optional().
			Nillable(),
	}
}

func (Plan) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("phases", PlanPhase.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
	}
}

func (Plan) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "key", "version", "deleted_at").
			Unique(),
	}
}

type PlanPhase struct {
	ent.Schema
}

func (PlanPhase) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.UniqueResourceMixin{},
	}
}

func (PlanPhase) Fields() []ent.Field {
	return []ent.Field{
		field.String("start_after").
			GoType(datex.ISOString("")).
			Default(plan.DefaultStartAfter),
		field.String("discounts").
			GoType([]plan.Discount{}).
			ValueScanner(DiscountsValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional(),
		field.String("plan_id").
			NotEmpty().
			Comment("The plan identifier the phase is assigned to."),
	}
}

func (PlanPhase) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("plan", Plan.Type).
			Ref("phases").
			Field("plan_id").
			Required().
			Unique(),
		edge.To("ratecards", PlanRateCard.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
	}
}

func (PlanPhase) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "key"),
		index.Fields("plan_id", "key").
			Unique(),
	}
}

type PlanRateCard struct {
	ent.Schema
}

func (PlanRateCard) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.UniqueResourceMixin{},
	}
}

func (PlanRateCard) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("type").
			GoType(plan.RateCardType("")).
			Immutable(),
		field.String("feature_key").
			Optional().
			Nillable(),
		field.String("entitlement_template").
			GoType(&plan.EntitlementTemplate{}).
			ValueScanner(EntitlementTemplateValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),
		field.String("tax_config").
			GoType(&plan.TaxConfig{}).
			ValueScanner(TaxConfigValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),
		field.String("billing_cadence").
			GoType(datex.ISOString("")).
			Optional().
			Nillable(),
		field.String("price").
			GoType(&plan.Price{}).
			ValueScanner(PriceValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),
		field.String("phase_id").
			NotEmpty().
			Comment("The phase identifier the ratecard is assigned to."),
		field.String("feature_id").
			Optional().
			Nillable().
			Comment("The feature identifier the ratecard is related to."),
	}
}

func (PlanRateCard) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("phase", PlanPhase.Type).
			Ref("ratecards").
			Field("phase_id").
			Required().
			Unique(),
		edge.From("features", Feature.Type).
			Ref("ratecard").
			Field("feature_id").
			Unique(),
	}
}

func (PlanRateCard) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("phase_id", "key").
			Unique(),
		index.Fields("phase_id", "feature_key").
			Unique(),
	}
}

var EntitlementTemplateValueScanner = entutils.JSONStringValueScanner[*plan.EntitlementTemplate]()

var TaxConfigValueScanner = entutils.JSONStringValueScanner[*plan.TaxConfig]()

var PriceValueScanner = entutils.JSONStringValueScanner[*plan.Price]()

var DiscountsValueScanner = entutils.JSONStringValueScanner[[]plan.Discount]()
