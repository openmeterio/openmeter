package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type UsageReset struct {
	ent.Schema
}

func (UsageReset) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.AnnotationsMixin{},
		entutils.TimeMixin{},
	}
}

func (UsageReset) Fields() []ent.Field {
	return []ent.Field{
		field.String("entitlement_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.Time("reset_time").Immutable(),
		field.Time("anchor").Immutable(),
		field.String("usage_period_interval").GoType(datetime.ISODurationString("")).Immutable(),
	}
}

func (UsageReset) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "entitlement_id"),
		index.Fields("namespace", "entitlement_id", "reset_time"),
	}
}

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
