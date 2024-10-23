package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type SubscriptionEntitlement struct {
	ent.Schema
}

func (SubscriptionEntitlement) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (SubscriptionEntitlement) Fields() []ent.Field {
	return []ent.Field{
		field.String("entitlement_id").NotEmpty().Immutable(),
		field.String("subscription_id").NotEmpty().Immutable(),
		field.String("subscription_phase_key").NotEmpty().Immutable(),
		field.String("subscription_item_key").NotEmpty().Immutable(),
	}
}

func (SubscriptionEntitlement) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "entitlement_id"),
		index.Fields("namespace", "subscription_id"),
		index.Fields("namespace", "subscription_id", "subscription_phase_key", "subscription_item_key"),
	}
}

func (SubscriptionEntitlement) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("subscription", Subscription.Type).Field("subscription_id").Ref("entitlements").Unique().Required().Immutable(),
		edge.From("entitlement", Entitlement.Type).Field("entitlement_id").Ref("subscription").Unique().Required().Immutable(),
	}
}
