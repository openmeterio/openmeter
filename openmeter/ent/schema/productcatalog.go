package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
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
		edge.To("subscriptions", Subscription.Type),
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
		field.String("plan_id").
			NotEmpty().
			Comment("The plan identifier the phase is assigned to."),
		field.Uint8("index").
			Comment("The index of the phase in the plan."),
		field.String("duration").
			GoType(datex.ISOString("")).
			Optional().
			Nillable().
			Comment("The duration of the phase."),
		field.String("discounts").
			GoType([]productcatalog.Discount{}).
			ValueScanner(DiscountsValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional(),
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
		index.Fields("plan_id", "key", "deleted_at").
			Unique(),
		index.Fields("plan_id", "index", "deleted_at").
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
			GoType(productcatalog.RateCardType("")).
			Immutable(),
		field.String("feature_key").
			Optional().
			Nillable(),
		field.String("entitlement_template").
			GoType(&productcatalog.EntitlementTemplate{}).
			ValueScanner(EntitlementTemplateValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),
		field.String("tax_config").
			GoType(&productcatalog.TaxConfig{}).
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
			GoType(&productcatalog.Price{}).
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
		index.Fields("phase_id", "key", "deleted_at").
			Unique(),
		index.Fields("phase_id", "feature_key", "deleted_at").
			Unique(),
	}
}

var (
	EntitlementTemplateValueScanner = entutils.JSONStringValueScanner[*productcatalog.EntitlementTemplate]()
	TaxConfigValueScanner           = entutils.JSONStringValueScanner[*productcatalog.TaxConfig]()
	PriceValueScanner               = entutils.JSONStringValueScanner[*productcatalog.Price]()
	DiscountsValueScanner           = entutils.JSONStringValueScanner[[]productcatalog.Discount]()
)
