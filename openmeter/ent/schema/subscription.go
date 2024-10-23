package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Subscription struct {
	ent.Schema
}

func (Subscription) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
		entutils.MetadataAnnotationsMixin{},
		entutils.CadencedMixin{},
	}
}

func (Subscription) Fields() []ent.Field {
	return []ent.Field{
		field.String("plan_key").NotEmpty().Immutable(),
		field.Int("plan_version").Min(1).Immutable(),
		field.String("customer_id").NotEmpty().Immutable(),
		field.String("currency").GoType(currencyx.Code("")).MinLen(3).MaxLen(3).NotEmpty().Immutable(),
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
		edge.To("subscription_patches", SubscriptionPatch.Type),
		edge.To("prices", Price.Type),
		edge.To("entitlements", SubscriptionEntitlement.Type),
		edge.From("customer", Customer.Type).Field("customer_id").Ref("subscription").Immutable().Unique().Required(),
	}
}
