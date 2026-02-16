package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
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

func (LedgerTransactionGroup) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("transactions", LedgerTransaction.Type),
	}
}

func (LedgerTransactionGroup) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id").Unique(),
	}
}
