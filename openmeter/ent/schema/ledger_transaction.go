package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type LedgerTransaction struct {
	ent.Schema
}

func (LedgerTransaction) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.AnnotationsMixin{},
		entutils.TimeMixin{},
	}
}

func (LedgerTransaction) Fields() []ent.Field {
	return []ent.Field{
		field.String("group_id").SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}).Immutable(),
		field.Time("booked_at").Immutable(),
	}
}

func (LedgerTransaction) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("group", LedgerTransactionGroup.Type).
			Ref("transactions").
			Field("group_id").
			Required().
			Immutable().
			Unique(),
		edge.To("entries", LedgerEntry.Type),
	}
}

func (LedgerTransaction) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id").Unique(),
		index.Fields("namespace", "group_id"),
		index.Fields("namespace", "booked_at"),
	}
}
