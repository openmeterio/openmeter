package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// EventOutbox holds system events until the broker acknowledges publication.
// Delivered rows are removed; this is a delivery queue, not an event archive.
type EventOutbox struct {
	ent.Schema
}

func (EventOutbox) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Time("created_at").Default(clock.Now).Immutable(),
		field.String("message_id").NotEmpty().Immutable(),
		field.String("transaction_id").NotEmpty().Immutable().SchemaType(map[string]string{dialect.Postgres: "char(26)"}),
		field.Int("attempts").Default(0).NonNegative(),
		field.String("topic").NotEmpty().Immutable(),
		field.Bytes("payload").Immutable(),
		field.String("metadata").
			GoType(map[string]string{}).
			ValueScanner(entutils.JSONStringValueScanner[map[string]string]()).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}).
			Immutable(),
	}
}

func (EventOutbox) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("topic", "id"),
		index.Fields("topic", "transaction_id", "id"),
	}
}
