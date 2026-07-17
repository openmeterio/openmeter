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
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Addon struct {
	ent.Schema
}

func (Addon) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.UniqueResourceMixin{},
	}
}

func (Addon) Fields() []ent.Field {
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
		field.Enum("instance_type").
			GoType(productcatalog.AddonInstanceType("")).
			Default(string(productcatalog.AddonInstanceTypeSingle)),
		field.Time("effective_from").
			Optional().
			Nillable(),
		field.Time("effective_to").
			Optional().
			Nillable(),
		field.String("annotations").
			GoType(models.Annotations{}).
			ValueScanner(AnnotationsValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional(),
	}
}

func (Addon) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("ratecards", AddonRateCard.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("plans", PlanAddon.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("subscription_addons", SubscriptionAddon.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.From("custom_currency", CustomCurrency.Type).
			Ref("addons").
			Field("custom_currency_id").
			Unique().
			Immutable(),
	}
}

func (Addon) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "key", "version").
			Annotations(
				entsql.IndexWhere("deleted_at IS NULL"),
			).
			Unique(),
		// GIN indexes can only be set on specific types such as jsonb
		index.Fields("annotations").
			Annotations(
				entsql.IndexTypes(map[string]string{
					dialect.Postgres: "GIN",
				}),
			),
		index.Fields("custom_currency_id"),
	}
}

func (Addon) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Checks(map[string]string{
			"addon_currency_reference": `(currency IS NULL) <> (custom_currency_id IS NULL)`,
		}),
	}
}

type AddonRateCard struct {
	ent.Schema
}

func (AddonRateCard) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.UniqueResourceMixin{},
		TaxMixin{},
	}
}

func (AddonRateCard) Fields() []ent.Field {
	fields := RateCard{}.Fields() // We have to use it like so due to some ent/runtime.go bug

	fields = append(fields,
		field.String("addon_id").
			NotEmpty().
			Comment("The add-on identifier the ratecard is assigned to."),
		field.String("feature_id").
			Optional().
			Nillable().
			Comment("The feature identifier the ratecard is related to."),
	)

	return fields
}

func (AddonRateCard) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("addon", Addon.Type).
			Ref("ratecards").
			Field("addon_id").
			Required().
			Unique(),
		edge.From("features", Feature.Type).
			Ref("addon_ratecard").
			Field("feature_id").
			Unique(),
		edge.From("tax_code", TaxCode.Type).
			Ref("addon_rate_cards").
			Field("tax_code_id").
			Unique(),
		edge.From("custom_currency", CustomCurrency.Type).
			Ref("addon_rate_cards").
			Field("custom_currency_id").
			Unique(),
	}
}

func (AddonRateCard) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("addon_id", "key").
			Annotations(
				entsql.IndexWhere("deleted_at IS NULL"),
			).
			Unique(),
		index.Fields("addon_id", "feature_key").
			Annotations(
				entsql.IndexWhere("deleted_at IS NULL"),
			).
			Unique(),
		index.Fields("custom_currency_id"),
	}
}

func (AddonRateCard) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Checks(map[string]string{
			"addon_rate_card_currency_reference": `currency IS NULL OR custom_currency_id IS NULL`,
			"addon_rate_card_currency_has_price": `price IS NOT NULL OR (currency IS NULL AND custom_currency_id IS NULL)`,
		}),
	}
}
