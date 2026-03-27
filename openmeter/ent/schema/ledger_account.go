package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

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

func (LedgerAccount) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("sub_accounts", LedgerSubAccount.Type),
		edge.To("sub_account_routes", LedgerSubAccountRoute.Type),
	}
}

type LedgerSubAccount struct {
	ent.Schema
}

func (LedgerSubAccount) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.AnnotationsMixin{},
		entutils.TimeMixin{},
	}
}

func (LedgerSubAccount) Fields() []ent.Field {
	return []ent.Field{
		field.String("account_id").SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}).Immutable(),
		field.String("route_id").SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}).Immutable(),
	}
}

func (LedgerSubAccount) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "account_id", "route_id").Unique(),
	}
}

func (LedgerSubAccount) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("account", LedgerAccount.Type).
			Ref("sub_accounts").
			Field("account_id").
			Required().
			Immutable().
			Unique(),
		edge.From("route", LedgerSubAccountRoute.Type).
			Ref("sub_accounts").
			Field("route_id").
			Required().
			Immutable().
			Unique(),
		edge.To("entries", LedgerEntry.Type),
	}
}

type LedgerSubAccountRoute struct {
	ent.Schema
}

func (LedgerSubAccountRoute) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (LedgerSubAccountRoute) Fields() []ent.Field {
	return []ent.Field{
		field.String("account_id").SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}).Immutable(),
		field.String("routing_key_version").
			GoType(ledger.RoutingKeyVersion("")).
			Immutable(),
		field.String("routing_key").Immutable(),
		// Literal routing values
		field.String("currency").Immutable(),
		field.String("tax_code").Optional().Nillable().Immutable(),
		field.Strings("features").Optional().Immutable(),
		field.Other("cost_basis", alpacadecimal.Decimal{}).
			Optional().Nillable().Immutable().
			SchemaType(map[string]string{dialect.Postgres: "numeric"}),
		field.Int("credit_priority").Optional().Nillable().Immutable(),
		field.String("transaction_authorization_status").
			GoType(ledger.TransactionAuthorizationStatus("")).
			Optional().Nillable().Immutable(),
	}
}

func (LedgerSubAccountRoute) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "account_id", "routing_key_version", "routing_key").Unique(),
	}
}

func (LedgerSubAccountRoute) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("account", LedgerAccount.Type).
			Ref("sub_account_routes").
			Field("account_id").
			Required().
			Immutable().
			Unique(),
		edge.To("sub_accounts", LedgerSubAccount.Type),
	}
}
