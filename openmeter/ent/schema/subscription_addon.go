package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// SubscriptionAddon is an instantiated addon for a subscription
type SubscriptionAddon struct {
	ent.Schema
}

func (SubscriptionAddon) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.MetadataMixin{},
		entutils.TimeMixin{},
	}
}

func (SubscriptionAddon) Fields() []ent.Field {
	return []ent.Field{
		field.String("addon_id").NotEmpty().Immutable(),
		field.String("subscription_id").NotEmpty().Immutable(),
	}
}

func (SubscriptionAddon) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("subscription", Subscription.Type).
			Ref("addons").
			Field("subscription_id").
			Unique().
			Required().
			Immutable(),
		edge.To("quantities", SubscriptionAddonQuantity.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.From("addon", Addon.Type).
			Ref("subscription_addons").
			Field("addon_id").
			Unique().
			Required().
			Immutable(),
	}
}

type SubscriptionAddonQuantity struct {
	ent.Schema
}

func (SubscriptionAddonQuantity) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (SubscriptionAddonQuantity) Fields() []ent.Field {
	return []ent.Field{
		field.Time("active_from").Default(clock.Now).Immutable(),
		field.Int("quantity").Default(1).Min(0).Immutable(),
		field.String("subscription_addon_id").NotEmpty().Immutable(),
	}
}

func (SubscriptionAddonQuantity) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("subscription_addon_id"),
	}
}

func (SubscriptionAddonQuantity) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("subscription_addon", SubscriptionAddon.Type).
			Ref("quantities").
			Field("subscription_addon_id").
			Required().
			Unique().
			Immutable(),
	}
}
