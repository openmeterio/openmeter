package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
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
		field.String("currency").
			Default("USD").
			NotEmpty().
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
	}
}

type AddonRateCard struct {
	ent.Schema
}

func (AddonRateCard) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.UniqueResourceMixin{},
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
		edge.To("subscription_addon_rate_cards", SubscriptionAddonRateCard.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
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
	}
}
