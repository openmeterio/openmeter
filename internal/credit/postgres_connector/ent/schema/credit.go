package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/internal/credit"
)

type CreditEntry struct {
	ent.Schema
}

// Mixin of the CreditGrant.
func (CreditEntry) Mixin() []ent.Mixin {
	return []ent.Mixin{
		IDMixin{},
		TimeMixin{},
	}
}

// Fields of the CreditGrant.
func (CreditEntry) Fields() []ent.Field {
	return []ent.Field{
		field.String("namespace").NotEmpty().Immutable(),
		field.String("ledger_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.Enum("entry_type").GoType(credit.EntryType("")).Immutable(),
		field.Enum("type").GoType(credit.GrantType("")).Optional().Nillable().Immutable(),
		field.String("feature_id").Optional().Nillable().Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.Float("amount").Optional().Nillable().Immutable().SchemaType(map[string]string{
			dialect.Postgres: "numeric",
		}),
		field.Uint8("priority").Default(1).Immutable(),
		field.Time("effective_at").Default(time.Now).Immutable(),
		// Expiration
		field.Enum("expiration_period_duration").GoType(credit.ExpirationPeriodDuration("")).Optional().Nillable().Immutable(),
		field.Uint8("expiration_period_count").Optional().Nillable().Immutable(),
		field.Time("expiration_at").Optional().Nillable().Immutable(),
		// Rollover
		field.Enum("rollover_type").GoType(credit.GrantRolloverType("")).Optional().Nillable().Immutable(),
		field.Float("rollover_max_amount").Optional().Nillable().Immutable().SchemaType(map[string]string{
			dialect.Postgres: "numeric",
		}),
		field.JSON("metadata", map[string]string{}).Optional(),
		// Rollover or void grants will have a parent_id
		field.String("parent_id").Optional().Nillable().Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
	}
}

// Indexes of the CreditGrant.
func (CreditEntry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "ledger_id"),
	}
}

// Edges of the CreditGrant define the relations to other entities.
func (CreditEntry) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("children", CreditEntry.Type).
			Unique().
			Annotations(entsql.OnDelete(entsql.Restrict)).
			From("parent").
			Unique().
			Immutable().
			Field("parent_id"),
		edge.From("feature", Feature.Type).
			Ref("credit_grants").
			Field("feature_id").
			Unique().
			Immutable(),
		edge.From("ledger", Ledger.Type).
			Ref("credit_grants").
			Field("ledger_id").
			Required().
			Unique().
			Immutable(),
	}
}
