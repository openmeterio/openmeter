package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// SubscriptionSyncStateV2 stores the last successful reconciliation. A missing
// row means that no successful v2 reconciliation has been recorded.
type SubscriptionSyncStateV2 struct{ ent.Schema }

func (SubscriptionSyncStateV2) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "subscription_sync_states_v2"}}
}

func (SubscriptionSyncStateV2) Mixin() []ent.Mixin {
	return []ent.Mixin{entutils.IDMixin{}, entutils.NamespaceMixin{}, entutils.TimeMixin{}}
}

func (SubscriptionSyncStateV2) Fields() []ent.Field {
	return []ent.Field{
		field.String("subscription_id").NotEmpty().Immutable().Unique().SchemaType(map[string]string{dialect.Postgres: "char(26)"}),
		field.Enum("last_success_state").GoType(subscriptionsync.SyncStateV2State("")),
		field.Time("last_success_at"),
	}
}

func (SubscriptionSyncStateV2) Indexes() []ent.Index {
	return []ent.Index{index.Fields("namespace", "subscription_id").Unique()}
}

func (SubscriptionSyncStateV2) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("subscription", Subscription.Type).Ref("sync_state_v2").Field("subscription_id").Required().Immutable().Unique(),
	}
}
