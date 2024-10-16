package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type SubscriptionPatch struct {
	ent.Schema
}

func (SubscriptionPatch) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
		entutils.MetadataAnnotationsMixin{},
	}
}

func (SubscriptionPatch) Fields() []ent.Field {
	return []ent.Field{
		field.String("subscription_id").NotEmpty().Immutable(),
		field.Time("applied_at").Immutable(),
		field.Int("batch_index").Immutable(),
		field.String("operation").NotEmpty().Immutable(),
		field.String("path").NotEmpty().Immutable(),
	}
}

func (SubscriptionPatch) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "subscription_id"),
	}
}

func (SubscriptionPatch) Edges() []ent.Edge {
	return []ent.Edge{
		// edge.To("subscription_id", Subscription.Type),
	}
}

// We create dedicated tables for each value type
// This way data migration problems between different versions of patches mostly become
// schema migration problems that we're staticly forced to address.

type SubscriptionPatchValueAddItem struct {
	ent.Schema
}

func (SubscriptionPatchValueAddItem) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
	}
}

func (SubscriptionPatchValueAddItem) Fields() []ent.Field {
	return []ent.Field{
		field.String("patch_value_type").GoType(subscription.PatchOperation("")).Immutable(),
		field.String("subscription_patch_id").NotEmpty().Immutable(),
		// Data properties
		// Add
		field.String("phase_key").NotEmpty().Immutable(),
		field.String("item_key").NotEmpty().Immutable(),
		field.String("feature_key").Optional().Nillable().Immutable(),
		field.String("create_entitlement_entitlement_type").Optional().Nillable().Immutable(),
		field.Time("create_entitlement_measure_usage_from").Optional().Nillable().Immutable(),
		field.Float("create_entitlement_issue_after_reset").Optional().Nillable().Immutable(),
		field.Int("create_entitlement_issue_after_reset_priority").Optional().Nillable().Immutable(),
		field.Bool("create_entitlement_is_soft_limit").Optional().Nillable().Immutable(),
		field.Bool("create_entitlement_preserve_overage_at_reset").Optional().Nillable().Immutable(),
		field.JSON("create_entitlement_config", []byte{}).SchemaType(map[string]string{
			dialect.Postgres: "jsonb",
		}).Optional().Immutable(),
		field.String("create_entitlement_usage_period_interval").Optional().Nillable().Immutable(),
		field.Time("create_entitlement_usage_period_anchor").Optional().Nillable().Immutable(),
		field.Float("create_price_value").SchemaType(map[string]string{
			dialect.Postgres: "numeric",
		}).Optional().Nillable().Immutable(),
	}
}

func (SubscriptionPatchValueAddItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "subscription_patch_id"),
	}
}

func (SubscriptionPatchValueAddItem) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("subscription_patch_id", SubscriptionPatch.Type),
	}
}

type SubscriptionPatchValueAddPhase struct {
	ent.Schema
}

func (SubscriptionPatchValueAddPhase) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
	}
}

func (SubscriptionPatchValueAddPhase) Fields() []ent.Field {
	return []ent.Field{
		field.String("patch_value_type").GoType(subscription.PatchOperation("")).Immutable(),
		field.String("subscription_patch_id").NotEmpty().Immutable(),
		// Data Properties
		// Add
		field.String("phase_key").NotEmpty().Immutable(),
		field.String("start_after_iso").SchemaType(map[string]string{
			dialect.Postgres: "interval",
		}).Optional().Nillable().Immutable(),
		field.String("create_discount").NotEmpty().Immutable(),
		field.Strings("create_discount_applies_to").Optional().Immutable(),
		// Extend
		// Extend
		field.String("start_after_iso").SchemaType(map[string]string{
			dialect.Postgres: "interval",
		}).Optional().Nillable().Immutable(),
	}
}

func (SubscriptionPatchValueAddPhase) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "subscription_patch_id"),
	}
}

func (SubscriptionPatchValueAddPhase) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("subscription_patch_id", SubscriptionPatch.Type),
	}
}
