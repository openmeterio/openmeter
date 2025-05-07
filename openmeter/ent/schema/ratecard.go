package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/isodate"
)

type RateCard struct {
	ent.Schema
}

func (RateCard) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.UniqueResourceMixin{},
		EntitlementTemplateMixin{},
	}
}

func (RateCard) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("type").
			GoType(productcatalog.RateCardType("")).
			Immutable(),
		field.String("feature_key").
			Optional().
			Nillable(),
		field.String("feature_id").
			Optional().
			Nillable(),
		// FIXME: we should normalize these fields as well
		field.String("tax_config").
			GoType(&productcatalog.TaxConfig{}).
			ValueScanner(TaxConfigValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),
		field.String("billing_cadence").
			GoType(isodate.String("")).
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
		field.String("discounts").
			GoType(&productcatalog.Discounts{}).
			ValueScanner(DiscountsValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),
	}
}

func (RateCard) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("addon_ratecard", AddonRateCard.Type).Unique(),
		edge.To("plan_ratecard", PlanRateCard.Type).Unique(),
		edge.To("subscription_item", SubscriptionItem.Type).Unique(),
		edge.From("feature", Feature.Type).
			Ref("ratecards").
			Field("feature_id").
			Unique(),
	}
}

// Mixins for RateCards
type EntitlementTemplateMixin struct {
	mixin.Schema
}

func (EntitlementTemplateMixin) Fields() []ent.Field {
	fields := []ent.Field{
		field.Enum("entitlement_type").
			Values(entitlement.EntitlementType("").StrValues()...).
			Immutable(),
		field.JSON("metadata", map[string]string{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
		field.Bool("is_soft_limit").Optional().Nillable().Immutable(),
		field.Float("issue_after_reset").Optional().Nillable().Immutable(),
		field.Uint8("issue_after_reset_priority").Optional().Nillable().Immutable(),
		field.Bool("preserve_overage_at_reset").Optional().Nillable().Immutable(),
		field.JSON("config", []byte{}).SchemaType(map[string]string{
			dialect.Postgres: "jsonb",
		}).Optional(),
		field.String("usage_period").Optional().Nillable().Immutable(),
	}

	// Let's add a field name prefix to all fields
	fields = lo.Map(fields, func(field ent.Field, _ int) ent.Field {
		field.Descriptor().Name = "entitlement_template_" + field.Descriptor().Name
		return field
	})

	return fields
}

type RateCardMixin struct {
	mixin.Schema
}

func (RateCardMixin) Fields() []ent.Field {
	// Name fields (name, description) and key field are missing as they're present in the UniqueResourceMixin...
	var fields []ent.Field

	fields = append(
		fields,
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
			GoType(isodate.String("")).
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
		field.String("discounts").
			GoType(&productcatalog.Discounts{}).
			ValueScanner(DiscountsValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),
	)

	return fields
}
