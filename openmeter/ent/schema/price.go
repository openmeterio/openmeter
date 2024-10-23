package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Price struct {
	ent.Schema
}

func (Price) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
		entutils.CadencedMixin{},
	}
}

func (Price) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").NotEmpty().Immutable(),
		field.String("subscription_id").NotEmpty().Immutable(),
		field.String("phase_key").NotEmpty().Immutable(),
		field.String("item_key").NotEmpty().Immutable(),
		field.String("value").SchemaType(map[string]string{
			"postgresql": "numeric",
		}).NotEmpty().Immutable(),
	}
}

func (Price) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "subscription_id"),
		index.Fields("namespace", "subscription_id", "key"),
	}
}

func (Price) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("subscription", Subscription.Type).Field("subscription_id").Ref("prices").Immutable().Unique().Required(),
	}
}
