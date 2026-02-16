package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type LedgerEntry struct {
	ent.Schema
}

func (LedgerEntry) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.AnnotationsMixin{},
		entutils.TimeMixin{},
	}
}

func (LedgerEntry) Fields() []ent.Field {
	return []ent.Field{
		field.String("sub_account_id").SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}).Immutable(),
		field.Other("amount", alpacadecimal.Decimal{}).Immutable().SchemaType(map[string]string{
			dialect.Postgres: "numeric",
		}),
		field.String("transaction_id").SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}).Immutable(),
	}
}

func (LedgerEntry) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("transaction", LedgerTransaction.Type).
			Ref("entries").
			Field("transaction_id").
			Required().
			Immutable().
			Unique(),
		edge.From("sub_account", LedgerSubAccount.Type).
			Ref("entries").
			Field("sub_account_id").
			Required().
			Immutable().
			Unique(),
	}
}

func (LedgerEntry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id").Unique(),
		index.Fields("namespace", "transaction_id"),
		index.Fields("namespace", "sub_account_id"),
		index.Fields("created_at", "id").Annotations(
			entsql.IndexWhere("deleted_at IS NULL"),
		),
	}
}
