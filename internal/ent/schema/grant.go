package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

type Grant struct {
	ent.Schema
}

func (Grant) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.MetadataAnnotationsMixin{},
		entutils.TimeMixin{},
	}
}

func (Grant) Fields() []ent.Field {
	return []ent.Field{
		field.String("owner_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.Float("amount").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "numeric",
		}),
		field.Uint8("priority").Default(0).Immutable(),
		field.Time("effective_at").Immutable(),
		field.JSON("expiration", credit.ExpirationPeriod{}).Immutable().SchemaType(map[string]string{
			dialect.Postgres: "jsonb",
		}),
		field.Time("expires_at").Immutable(),
		field.Time("voided_at").Optional().Nillable(),
		field.Float("reset_max_rollover").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "numeric",
		}),
		field.Float("reset_min_rollover").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "numeric",
		}),
		field.Enum("recurrence_period").Optional().Nillable().GoType(recurrence.RecurrenceInterval("")).Immutable(),
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
