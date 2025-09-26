package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Grant struct {
	ent.Schema
}

func (Grant) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.AnnotationsMixin{},
		entutils.TimeMixin{},
	}
}

func (Grant) Fields() []ent.Field {
	return []ent.Field{
		field.JSON("metadata", map[string]string{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
		field.String("owner_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.Float("amount").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "numeric",
		}),
		field.Uint8("priority").Default(0).Immutable(),
		field.Time("effective_at").Immutable(),
		field.JSON("expiration", &grant.ExpirationPeriod{}).Optional().Immutable().SchemaType(map[string]string{
			dialect.Postgres: "jsonb",
		}),
		field.Time("expires_at").Optional().Nillable().Immutable(),
		field.Time("voided_at").Optional().Nillable(),
		field.Float("reset_max_rollover").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "numeric",
		}),
		field.Float("reset_min_rollover").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "numeric",
		}),
		field.String("recurrence_period").Optional().Nillable().GoType(datetime.ISODurationString("")).Immutable(),
		field.Time("recurrence_anchor").Optional().Nillable().Immutable(),
	}
}

func (Grant) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "owner_id"),
		index.Fields("effective_at", "expires_at"),
	}
}

func (Grant) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("entitlement", Entitlement.Type).
			Ref("grant").
			Field("owner_id").
			Required().
			Immutable().
			Unique(),
	}
}
