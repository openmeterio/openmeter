package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"
	"github.com/jackc/pgtype"

	"github.com/openmeterio/openmeter/openmeter/ledger"
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
		field.String("account_id").SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}).Immutable(),
		field.String("account_type").GoType(ledger.AccountType("")).Immutable(),
		// NOTE: We use pgtype.TextArray here instead of field.Strings because, in
		// our current Ent version, field.Strings encode/decode behavior does not
		// align with a native Postgres text[] column.
		//
		// Exact example of what does *not* matter: setting
		// SchemaType(map[string]string{dialect.Postgres: "text[]"})
		// only changes the generated column type in migrations/DDL. It does not
		// change field.Strings runtime serialization, so writes can still look like
		// JSON-style values (e.g. ["d1","d2"]) instead of native text[] values
		// (e.g. {"d1","d2"}).
		// e.g. failed to create ledger entries: insert nodes to table "ledger_entries": ERROR: malformed array literal: "["01KH455F5DW134P67193Q1TWQN","01KH455F5NJ8W1690NFVPA5W22","01KH455F5NJ8W1690NFXW2XTV0"]" (SQLSTATE 22P02)
		field.Other("dimension_ids", pgtype.TextArray{}).Optional().SchemaType(map[string]string{
			dialect.Postgres: "text[]",
		}),
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
	}
}

func (LedgerEntry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id").Unique(),
		index.Fields("namespace", "transaction_id"),
		index.Fields("namespace", "account_id"),
		index.Fields("created_at", "id").Annotations(
			entsql.IndexWhere("deleted_at IS NULL"),
		),
	}
}
