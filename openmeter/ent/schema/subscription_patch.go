package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type SubscriptionPatch struct {
	ent.Schema
}

func (SubscriptionPatch) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "subscription_patches"},
	}
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
		edge.From("subscription", Subscription.Type).Ref("subscription_patches").Field("subscription_id").Unique().Immutable().Required(),
		edge.To("value_add_item", SubscriptionPatchValueAddItem.Type).Unique(),
		edge.To("value_add_phase", SubscriptionPatchValueAddPhase.Type).Unique(),
		edge.To("value_extend_phase", SubscriptionPatchValueExtendPhase.Type).Unique(),
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
		field.String("subscription_patch_id").NotEmpty().Immutable(),
		// Data properties
		field.String("phase_key").NotEmpty().Immutable(),
		field.String("item_key").NotEmpty().Immutable(),
		field.String("feature_key").Optional().Nillable().Immutable(),
		field.String("create_entitlement_entitlement_type").Optional().Nillable().Immutable(),
		field.Float("create_entitlement_issue_after_reset").Optional().Nillable().Immutable(),
		field.Uint8("create_entitlement_issue_after_reset_priority").Optional().Nillable().Immutable(),
		field.Bool("create_entitlement_is_soft_limit").Optional().Nillable().Immutable(),
		field.Bool("create_entitlement_preserve_overage_at_reset").Optional().Nillable().Immutable(),
		field.String("create_entitlement_usage_period_iso_duration").Optional().Nillable().Immutable(),
		field.JSON("create_entitlement_config", []byte{}).SchemaType(map[string]string{
			dialect.Postgres: "jsonb",
		}).Optional().Immutable(),
		field.String("create_price_key").Optional().Nillable().Immutable(),
		field.String("create_price_value").SchemaType(map[string]string{
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
		edge.From("subscription_patch", SubscriptionPatch.Type).Ref("value_add_item").Field("subscription_patch_id").Immutable().Unique().Required(),
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
		field.String("subscription_patch_id").NotEmpty().Immutable(),
		// Data Properties
		field.String("phase_key").NotEmpty().Immutable(),
		field.String("start_after_iso").Immutable(),
		field.Bool("create_discount").Immutable(),
		field.Strings("create_discount_applies_to").Optional().Immutable(),
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
		edge.From("subscription_patch", SubscriptionPatch.Type).Ref("value_add_phase").Field("subscription_patch_id").Immutable().Unique().Required(),
	}
}

type SubscriptionPatchValueExtendPhase struct {
	ent.Schema
}

func (SubscriptionPatchValueExtendPhase) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
	}
}

func (SubscriptionPatchValueExtendPhase) Fields() []ent.Field {
	return []ent.Field{
		field.String("subscription_patch_id").NotEmpty().Immutable(),
		// Data Properties
		field.String("phase_key").NotEmpty().Immutable(),
		field.String("extend_duration_iso").Immutable(),
	}
}

func (SubscriptionPatchValueExtendPhase) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "subscription_patch_id"),
	}
}

func (SubscriptionPatchValueExtendPhase) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("subscription_patch", SubscriptionPatch.Type).Ref("value_extend_phase").Field("subscription_patch_id").Immutable().Unique().Required(),
	}
}
