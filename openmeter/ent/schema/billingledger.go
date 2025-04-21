package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type BillingLedger struct {
	ent.Schema
}

func (BillingLedger) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (BillingLedger) Fields() []ent.Field {
	return []ent.Field{
		field.String("customer_id").
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.String("currency").
			GoType(currencyx.Code("")).
			Immutable(),
	}
}

func (BillingLedger) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "customer_id", "currency").
			Annotations(
				entsql.IndexWhere("deleted_at IS NULL"),
			).
			Unique(),
	}
}

func (BillingLedger) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("subledgers", BillingSubledger.Type),
		edge.To("transactions", BillingSubledgerTransaction.Type),
		edge.From("customer", Customer.Type).
			Ref("billing_ledger").
			Field("customer_id").
			Required().
			Immutable().
			Unique(),
	}
}

type BillingSubledger struct {
	ent.Schema
}

func (BillingSubledger) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.ResourceMixin{},
		entutils.KeyMixin{},
	}
}

func (BillingSubledger) Fields() []ent.Field {
	return []ent.Field{
		field.String("ledger_id").
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.Int64("priority").
			Default(0),
		// TODO: Tax app type
		// TODO: Resolved code for the app
	}
}

func (BillingSubledger) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "ledger_id", "key").
			Annotations(
				entsql.IndexWhere("deleted_at IS NULL"),
			).
			Unique(),
	}
}

func (BillingSubledger) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ledger", BillingLedger.Type).
			Ref("subledgers").
			Field("ledger_id").
			Required().
			Immutable().
			Unique(),
		edge.To("transactions", BillingSubledgerTransaction.Type),
	}
}

type BillingSubledgerTransaction struct {
	ent.Schema
}

func (BillingSubledgerTransaction) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.ResourceMixin{},
	}
}

func (BillingSubledgerTransaction) Fields() []ent.Field {
	return []ent.Field{
		field.String("subledger_id").
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.String("ledger_id").
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.Other("amount", alpacadecimal.Decimal{}).
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.String("owner_type").
			Optional().
			Nillable(),
		field.String("owner_id").
			Optional().
			Nillable(),
	}
}

func (BillingSubledgerTransaction) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("subledger_id"),
		index.Fields("ledger_id"),
	}
}

func (BillingSubledgerTransaction) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("subledger", BillingSubledger.Type).
			Ref("transactions").
			Field("subledger_id").
			Required().
			Immutable().
			Unique(),
		edge.From("ledger", BillingLedger.Type).
			Ref("transactions").
			Field("ledger_id").
			Required().
			Immutable().
			Unique(),
	}
}
