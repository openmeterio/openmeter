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
		edge.To("rate_cards", SubscriptionAddonRateCard.Type),
		edge.To("quantities", SubscriptionAddonQuantity.Type),
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

type SubscriptionAddonRateCard struct {
	ent.Schema
}

func (SubscriptionAddonRateCard) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.UniqueResourceMixin{},
	}
}

func (SubscriptionAddonRateCard) Fields() []ent.Field {
	var fields []ent.Field
	fields = append(fields,
		field.String("subscription_addon_id").NotEmpty().Immutable(),
		field.String("addon_ratecard_id").NotEmpty().Immutable(),
	)

	return fields
}

func (SubscriptionAddonRateCard) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("subscription_addon", SubscriptionAddon.Type).
			Ref("rate_cards").
			Field("subscription_addon_id").
			Required().
			Unique().
			Immutable(),
		edge.To("items", SubscriptionAddonRateCardItemLink.Type),
		edge.From("addon_ratecard", AddonRateCard.Type).
			Ref("subscription_addon_rate_cards").
			Field("addon_ratecard_id").
			Unique().
			Required().
			Immutable(),
	}
}

func (SubscriptionAddonRateCard) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("subscription_addon_id"),
	}
}

// Linking table between SubscriptionAddonRateCard and SubscriptionItem
type SubscriptionAddonRateCardItemLink struct {
	ent.Schema
}

func (SubscriptionAddonRateCardItemLink) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.TimeMixin{},
	}
}

func (SubscriptionAddonRateCardItemLink) Fields() []ent.Field {
	return []ent.Field{
		field.String("subscription_addon_rate_card_id").NotEmpty().Immutable(),
		field.String("subscription_item_id").NotEmpty().Immutable(),
	}
}

func (SubscriptionAddonRateCardItemLink) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("subscription_addon_rate_card_id"),
		index.Fields("subscription_item_id"),
		index.Fields("subscription_item_id", "subscription_addon_rate_card_id").
			Annotations(entsql.IndexWhere("deleted_at IS NULL")).
			Unique(), // To avoid duplicates
	}
}

func (SubscriptionAddonRateCardItemLink) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("subscription_addon_rate_card", SubscriptionAddonRateCard.Type).
			Ref("items").
			Field("subscription_addon_rate_card_id").
			Required().
			Unique().
			Immutable(),
		edge.From("subscription_item", SubscriptionItem.Type).
			Ref("subscription_addon_rate_card_items").
			Field("subscription_item_id").
			Required().
			Unique().
			Immutable(),
	}
}
