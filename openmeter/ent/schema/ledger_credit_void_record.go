package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type LedgerCreditVoidRecord struct {
	ent.Schema
}

// Credit void records are projection rows for customer-visible void events.
// Ledger entries remain the accounting source of truth.
func (LedgerCreditVoidRecord) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.AnnotationsMixin{},
		entutils.TimeMixin{},
	}
}

func (LedgerCreditVoidRecord) Fields() []ent.Field {
	return []ent.Field{
		field.Other("amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.String("customer_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			NotEmpty().
			Immutable(),
		field.String("currency").
			GoType(currencyx.Code("")).
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "varchar(3)",
			}),
		field.Time("voided_at").
			Immutable(),
		field.String("source_charge_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			NotEmpty().
			Immutable(),
		field.String("void_transaction_group_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			NotEmpty().
			Immutable(),
		field.String("void_transaction_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			NotEmpty().
			Immutable(),
		field.String("fbo_sub_account_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			NotEmpty().
			Immutable(),
		field.String("receivable_sub_account_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			NotEmpty().
			Immutable(),
	}
}

func (LedgerCreditVoidRecord) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "customer_id", "currency", "voided_at", "id").
			StorageKey("ledgercreditvoidrecord_namespace_customer_currency_voided"),
		index.Fields("namespace", "source_charge_id"),
		index.Fields("namespace", "void_transaction_group_id"),
	}
}
