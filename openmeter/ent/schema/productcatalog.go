package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/datetime"
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
		field.String("fiat_currency_code").
			StorageKey("currency").
			NotEmpty().
			MinLen(3).
			MaxLen(3).
			Optional().
			Nillable().
			Immutable(),
		field.String("custom_currency_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			NotEmpty().
			Optional().
			Nillable().
			Immutable(),
		field.String("billing_cadence").
			GoType(datetime.ISODurationString("")).
			Comment("The default billing cadence for subscriptions using this plan."),
		field.String("pro_rating_config").
			GoType(productcatalog.ProRatingConfig{}).
			ValueScanner(ProRatingConfigValueScanner).
			DefaultFunc(func() productcatalog.ProRatingConfig {
				return productcatalog.ProRatingConfig{
					Mode:    productcatalog.ProRatingModeProratePrices,
					Enabled: true,
				}
			}).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Comment("Default pro-rating configuration for subscriptions using this plan."),
		field.Time("effective_from").
			Optional().
			Nillable(),
		field.Time("effective_to").
			Optional().
			Nillable(),
		field.Enum("settlement_mode").
			GoType(productcatalog.SettlementMode("")).
			Default(string(productcatalog.CreditThenInvoiceSettlementMode)),
	}
}

func (Plan) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("phases", PlanPhase.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("addons", PlanAddon.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("subscriptions", Subscription.Type),
		edge.From("custom_currency", CustomCurrency.Type).
			Ref("plans").
			Field("custom_currency_id").
			Unique().
			Immutable(),
	}
}

func (Plan) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "key", "version").
			Annotations(
				entsql.IndexWhere("deleted_at IS NULL"),
			).
			Unique(),
		index.Fields("custom_currency_id"),
	}
}

func (Plan) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Checks(map[string]string{
			"plan_currency_reference": `(currency IS NULL) <> (custom_currency_id IS NULL)`,
		}),
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
			GoType(datetime.ISODurationString("")).
			Optional().
			Nillable().
			Comment("The duration of the phase."),
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
			Annotations(
				entsql.IndexWhere("deleted_at IS NULL"),
			).
			Unique(),
		index.Fields("plan_id", "index").
			Annotations(
				entsql.IndexWhere("deleted_at IS NULL"),
			).
			Unique(),
	}
}

type PlanRateCard struct {
	ent.Schema
}

func (PlanRateCard) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.UniqueResourceMixin{},
		TaxMixin{},
	}
}

func (PlanRateCard) Fields() []ent.Field {
	fields := RateCard{}.Fields() // We have to use it like so due to some ent/runtime.go bug

	fields = append(fields,
		field.String("phase_id").
			NotEmpty().
			Comment("The phase identifier the ratecard is assigned to."),
		field.String("feature_id").
			Optional().
			Nillable().
			Comment("The feature identifier the ratecard is related to."),
	)

	return fields
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
		edge.From("tax_code", TaxCode.Type).
			Ref("plan_rate_cards").
			Field("tax_code_id").
			Unique(),
		edge.From("custom_currency", CustomCurrency.Type).
			Ref("plan_rate_cards").
			Field("custom_currency_id").
			Unique(),
	}
}

func (PlanRateCard) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("phase_id", "key").
			Annotations(
				entsql.IndexWhere("deleted_at IS NULL"),
			).
			Unique(),
		index.Fields("phase_id", "feature_key").
			Annotations(
				entsql.IndexWhere("deleted_at IS NULL"),
			).
			Unique(),
		index.Fields("custom_currency_id"),
	}
}

func (PlanRateCard) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Checks(map[string]string{
			"plan_rate_card_currency_reference": `currency IS NULL OR custom_currency_id IS NULL`,
			"plan_rate_card_currency_has_price": `price IS NOT NULL OR (currency IS NULL AND custom_currency_id IS NULL)`,
		}),
	}
}

var (
	EntitlementTemplateValueScanner = entutils.JSONStringValueScanner[*productcatalog.EntitlementTemplate]()
	TaxConfigValueScanner           = entutils.JSONStringValueScanner[*productcatalog.TaxConfig]()
	PriceValueScanner               = entutils.JSONStringValueScanner[*productcatalog.Price]()
	DiscountsValueScanner           = entutils.JSONStringValueScanner[*productcatalog.Discounts]()
	UnitConfigValueScanner          = entutils.JSONStringValueScanner[*productcatalog.UnitConfig]()
	ProRatingConfigValueScanner     = entutils.JSONStringValueScanner[productcatalog.ProRatingConfig]()
)
