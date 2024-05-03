package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	credit_model "github.com/openmeterio/openmeter/internal/credit"
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
		field.String("subject").NotEmpty().Immutable(),
		field.Enum("entry_type").GoType(credit_model.EntryType("")).Immutable(),
		field.Enum("type").GoType(credit_model.GrantType("")).Optional().Nillable().Immutable(),
		field.String("feature_id").Optional().Nillable().Immutable(),
		// TODO: use decimal instead of float?
		field.Float("amount").Optional().Nillable().Immutable(),
		field.Uint8("priority").Default(1).Immutable(),
		field.Time("effective_at").Default(time.Now).Immutable(),
		// Expiration
		field.Enum("expiration_period_duration").GoType(credit_model.ExpirationPeriodDuration("")).Optional().Nillable().Immutable(),
		field.Uint8("expiration_period_count").Optional().Nillable().Immutable(),
		field.Time("expiration_at").Optional().Nillable().Immutable(),
		// Rollover
		field.Enum("rollover_type").GoType(credit_model.GrantRolloverType("")).Optional().Nillable().Immutable(),
		field.Float("rollover_max_amount").Optional().Nillable().Immutable(),
		field.JSON("metadata", map[string]string{}).Optional(),
		// Rollover or void grants will have a parent_id
		field.String("parent_id").Optional().Nillable().Immutable(),
	}
}

// Indexes of the CreditGrant.
func (CreditEntry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "subject"),
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
	}
}
