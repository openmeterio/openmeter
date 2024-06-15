package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type UsageReset struct {
	ent.Schema
}

// Mixin of the CreditGrant.
func (UsageReset) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{}, // won't be used
		entutils.NamespaceMixin{},
		entutils.TimeMixin{}, // maybe we don't need this
	}
}

// Fields of the CreditGrant.
func (UsageReset) Fields() []ent.Field {
	return []ent.Field{
		field.String("entitlement_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.Time("reset_time").Immutable(),
	}
}

// Indexes of the UsageReset.
// TODO: Atlas support will add the possibility to set dialect specific index implementations
func (UsageReset) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "entitlement_id"),
		index.Fields("namespace", "entitlement_id", "reset_time"),
	}
}

// Edges of the UsageReset define the relations to other entities.
// TOOD: link to entitlements
func (UsageReset) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("entitlement", Entitlement.Type).
			Ref("usage_reset").
			Field("entitlement_id").
			Required().
			Unique().
			Immutable(),
	}
}
