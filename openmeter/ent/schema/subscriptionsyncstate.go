package schema

import (
	"slices"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// SubscriptionSyncState stores subscription revisions and the last sync result.
// New rows have no last_success_at until a sync result is recorded.
//
// The table, its indexes and its constraints keep the names they got when this
// entity was called SubscriptionSyncStateV2, so renaming the entity does not
// rename anything in the database.
type SubscriptionSyncState struct{ ent.Schema }

func (SubscriptionSyncState) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "subscription_sync_states_v2"},
		// A missing deadline would silently remove an active subscription from reconciliation.
		entsql.Checks(map[string]string{
			"subscription_sync_v2_active_reconciliation_at": "last_success_state <> 'active' OR next_reconciliation_at IS NOT NULL",
		}),
	}
}

func (SubscriptionSyncState) Mixin() []ent.Mixin {
	return []ent.Mixin{subscriptionSyncStateIDMixin{}, entutils.TimeMixin{}}
}

func (SubscriptionSyncState) Fields() []ent.Field {
	return []ent.Field{
		field.String("subscription_id").NotEmpty().Immutable().Unique().SchemaType(map[string]string{dialect.Postgres: "char(26)"}),
		field.Enum("last_success_state").GoType(subscriptionsync.SyncStateState("")),
		field.Time("last_success_at").Optional().Nillable(),
		field.Time("next_reconciliation_at").Optional().Nillable(),
		field.String("last_synced_subs_revision").NotEmpty().SchemaType(map[string]string{dialect.Postgres: "char(26)"}),
		field.String("current_subs_revision").NotEmpty().SchemaType(map[string]string{dialect.Postgres: "char(26)"}),
	}
}

func (SubscriptionSyncState) Indexes() []ent.Index {
	return []ent.Index{
		// These two replace the IDMixin and NamespaceMixin indexes under their old names.
		index.Fields("id").Unique().StorageKey("subscriptionsyncstatev2_id"),
		index.Fields("namespace").StorageKey("subscriptionsyncstatev2_namespace"),
		// Include the result so deleted-subscription polling can check completed
		// cleanup from this index without loading the sync-state rows.
		index.Fields("namespace", "subscription_id").Unique().
			StorageKey("subscriptionsyncstatev2_namespace_subscription_id").
			Annotations(entsql.IncludeColumns("last_success_state")),
	}
}

func (SubscriptionSyncState) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("subscription", Subscription.Type).Ref("sync_state").Field("subscription_id").Required().Immutable().Unique(),
	}
}

// subscriptionSyncStateIDMixin has the IDMixin and NamespaceMixin fields
// without their indexes. Those mixins name their indexes after the entity, so
// SubscriptionSyncState declares the indexes itself with the old names.
type subscriptionSyncStateIDMixin struct{ mixin.Schema }

func (subscriptionSyncStateIDMixin) Fields() []ent.Field {
	return slices.Concat(entutils.IDMixin{}.Fields(), entutils.NamespaceMixin{}.Fields())
}
