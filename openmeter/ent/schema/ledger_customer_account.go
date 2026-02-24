package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// LedgerCustomerAccount is a private linking table that maps a customer to their
// ledger accounts (one FBO and one Receivable per customer per namespace).
type LedgerCustomerAccount struct {
	ent.Schema
}

func (LedgerCustomerAccount) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (LedgerCustomerAccount) Fields() []ent.Field {
	return []ent.Field{
		field.String("customer_id").Immutable(),
		field.String("account_type").GoType(ledger.AccountType("")).Immutable(),
		field.String("account_id").Immutable(),
	}
}

func (LedgerCustomerAccount) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id").Unique(),
		// One FBO and one Receivable account per customer per namespace
		index.Fields("namespace", "customer_id", "account_type").Unique(),
	}
}

func (LedgerCustomerAccount) Edges() []ent.Edge {
	return nil
}
