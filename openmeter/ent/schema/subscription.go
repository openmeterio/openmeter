package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Subscription struct {
	ent.Schema
}

func (Subscription) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
		entutils.MetadataMixin{},
		entutils.CadencedMixin{},
		AlignmentMixin{},
	}
}

func (Subscription) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty().Default("Subscription"),
		field.String("description").Optional().Nillable(),
		field.String("plan_id").Optional().Nillable(),
		field.String("customer_id").NotEmpty().Immutable(),
		field.String("currency").GoType(currencyx.Code("")).MinLen(3).MaxLen(3).NotEmpty().Immutable(),
		field.Time("billing_anchor"),
		field.String("billing_cadence").
			GoType(isodate.String("")).
			Comment("The default billing cadence for subscriptions."),
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
			Comment("Default pro-rating configuration for subscriptions."),
	}
}

func (Subscription) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "customer_id"),
	}
}

func (Subscription) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("plan", Plan.Type).Field("plan_id").Ref("subscriptions").Unique(),
		edge.From("customer", Customer.Type).Field("customer_id").Ref("subscription").Immutable().Unique().Required(),
		edge.To("phases", SubscriptionPhase.Type).Annotations(entsql.Annotation{
			OnDelete: entsql.Cascade,
		}),
		edge.To("billing_lines", BillingInvoiceLine.Type),
		edge.To("billing_split_line_groups", BillingInvoiceSplitLineGroup.Type),
		edge.To("addons", SubscriptionAddon.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
	}
}

type SubscriptionPhase struct {
	ent.Schema
}

func (SubscriptionPhase) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
		entutils.MetadataMixin{},
	}
}

func (SubscriptionPhase) Fields() []ent.Field {
	return []ent.Field{
		field.String("subscription_id").NotEmpty().Immutable(),
		field.String("key").NotEmpty().Immutable(),
		field.String("name").NotEmpty(),
		field.String("description").Optional().Nillable(),
		field.Time("active_from").Immutable(),
		field.Uint8("sort_hint").Optional().Nillable().Comment("Used to sort phases when they have the same active_from time (happens for 0 length phases)"),
	}
}

func (SubscriptionPhase) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "subscription_id"),
		index.Fields("namespace", "subscription_id", "key"),
	}
}

func (SubscriptionPhase) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("subscription", Subscription.Type).Field("subscription_id").Ref("phases").Unique().Immutable().Required(),
		edge.To("items", SubscriptionItem.Type).Annotations(entsql.Annotation{
			OnDelete: entsql.Cascade,
		}),
		edge.To("billing_lines", BillingInvoiceLine.Type),
		edge.To("billing_split_line_groups", BillingInvoiceSplitLineGroup.Type),
	}
}

type SubscriptionItem struct {
	ent.Schema
}

func (SubscriptionItem) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
		entutils.MetadataMixin{},
	}
}

func (SubscriptionItem) Fields() []ent.Field {
	return []ent.Field{
		field.String("annotations").
			GoType(models.Annotations{}).
			ValueScanner(AnnotationsValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional(),
		// Note: we cannot use CadencedMixin as these are mutable
		field.Time("active_from"),
		field.Time("active_to").Optional().Nillable(),
		field.String("phase_id").NotEmpty().Immutable(),
		field.String("key").NotEmpty().Immutable(),
		field.String("entitlement_id").Optional().Nillable(),
		field.Bool("restarts_billing_period").Optional().Nillable(),
		// Items can have different intended cadence compared to the phase due to edits.
		// To preserve this across cancels and other complex scenarios, we store the intended cadence relative to the phase start.
		field.String("active_from_override_relative_to_phase_start").
			GoType(isodate.String("")).Nillable().Optional(),
		field.String("active_to_override_relative_to_phase_start").
			GoType(isodate.String("")).Nillable().Optional(),
		// RateCard Fields
		field.String("name").NotEmpty(),
		field.String("description").Optional().Nillable(),
		field.String("feature_key").Optional().Nillable(),
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
	}
}

func (SubscriptionItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "phase_id", "key"),
	}
}

func (SubscriptionItem) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("phase", SubscriptionPhase.Type).Field("phase_id").Ref("items").Unique().Immutable().Required(),
		edge.From("entitlement", Entitlement.Type).Field("entitlement_id").Ref("subscription_item").Unique().Annotations(entsql.Annotation{
			OnDelete: entsql.Cascade,
		}),
		edge.To("billing_lines", BillingInvoiceLine.Type),
		edge.To("billing_split_line_groups", BillingInvoiceSplitLineGroup.Type),
	}
}
