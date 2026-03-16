package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
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
		field.String("currency").
			GoType(currencyx.Code("")).
			SchemaType(map[string]string{
				dialect.Postgres: "char(3)",
			}).
			Optional().
			NotEmpty().
			Immutable(),
		field.String("tax_code_id").SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}).
			Optional().
			NotEmpty().
			Immutable(),
		// TODO: implement feature keys
		field.String("feature_keys").SchemaType(map[string]string{
			dialect.Postgres: "text[]",
		}).Optional().Immutable(),
		field.Int("credit_priority").SchemaType(map[string]string{
			dialect.Postgres: "integer",
		}).Optional().Immutable(),
	}
}

func (LedgerSubAccountRoute) Indexes() []ent.Index {
	// TODO: have proper uniqeness constrains (e.g. coalesce)
	return []ent.Index{
		// index.Fields("namespace", "account_id", "routing_key_version", "routing_key").Unique(),

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
