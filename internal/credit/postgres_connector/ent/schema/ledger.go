package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

var defaultHighwatermark, _ = time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")

type Ledger struct {
	ent.Schema
}

// Mixin of the Ledger.
func (Ledger) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
		IDMixin{},
	}
}

// Fields of the Ledger.
func (Ledger) Fields() []ent.Field {
	return []ent.Field{
		field.String("namespace").NotEmpty().Immutable(),
		field.String("subject").NotEmpty().Immutable(),
		field.JSON("metadata", map[string]string{}).Optional(),
		field.Time("highwatermark").Default(func() time.Time {
			return defaultHighwatermark
		}),
	}
}

// Indexes of the Ledger.
func (Ledger) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "subject").Unique(),
	}
}
