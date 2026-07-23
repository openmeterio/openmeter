package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type LedgerTransactionGroup struct {
	ent.Schema
}

func (LedgerTransactionGroup) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.AnnotationsMixin{},
		entutils.TimeMixin{},
	}
}

func (LedgerTransactionGroup) Fields() []ent.Field {
	return []ent.Field{
		field.String("idempotency_key").
			Optional().
			Nillable().
			Immutable().
			MaxLen(256),
		field.String("input_fingerprint").
			Optional().
			Nillable().
			Immutable().
			MaxLen(67),
	}
}

func (LedgerTransactionGroup) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Checks(map[string]string{
			"ledger_tx_group_idempotency_pair": "(idempotency_key IS NULL) = (input_fingerprint IS NULL)",
		}),
	}
}

func (LedgerTransactionGroup) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("transactions", LedgerTransaction.Type),
		edge.To("source_breakage_records", LedgerBreakageRecord.Type).
			Annotations(entsql.OnDelete(entsql.Restrict)),
		edge.To("breakage_records", LedgerBreakageRecord.Type),
	}
}

func (LedgerTransactionGroup) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id").Unique(),
		index.Fields("namespace", "idempotency_key").
			Unique().
			Annotations(entsql.IndexWhere("idempotency_key IS NOT NULL")).
			StorageKey("ledger_tx_groups_namespace_idempotency_key"),
	}
}
