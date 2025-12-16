package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type SubscriptionBillingSyncState struct {
	ent.Schema
}

func (SubscriptionBillingSyncState) Fields() []ent.Field {
	return []ent.Field{
		field.String("subscription_id").
			NotEmpty().
			Immutable().
			Unique().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.Bool("has_billables"),
		field.Time("synced_at"),
		field.Time("next_sync_after").Optional().Nillable(),
	}
}

func (SubscriptionBillingSyncState) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
	}
}

func (SubscriptionBillingSyncState) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "subscription_id").Unique(),
	}
}

func (SubscriptionBillingSyncState) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("subscription", Subscription.Type).
			Ref("billing_sync_state").
			Field("subscription_id").
			Required().
			Immutable().
			Unique(),
	}
}
