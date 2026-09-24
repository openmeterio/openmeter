package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// EventOutbox holds system events until the broker acknowledges publication.
// Delivered rows are removed; this is a delivery queue, not an event archive.
type EventOutbox struct {
	ent.Schema
}

func (EventOutbox) Mixin() []ent.Mixin {
	return []ent.Mixin{entutils.TimeMixin{}}
}

func (EventOutbox) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("message_id").NotEmpty().Immutable(),
		field.String("topic").NotEmpty().Immutable(),
		field.String("delivery_key").Immutable(),
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
		index.Fields("topic", "id").Annotations(entsql.IndexWhere("deleted_at IS NULL")),
		index.Fields("topic", "delivery_key", "id").Annotations(entsql.IndexWhere("deleted_at IS NULL")),
	}
}
