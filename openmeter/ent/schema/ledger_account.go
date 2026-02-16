package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type LedgerAccount struct {
	ent.Schema
}

func (LedgerAccount) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.AnnotationsMixin{},
		entutils.TimeMixin{},
	}
}

func (LedgerAccount) Fields() []ent.Field {
	return []ent.Field{
		field.String("account_type").GoType(ledger.AccountType("")).Immutable(),
	}
}

func (LedgerAccount) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id").Unique(),
	}
}
