package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
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

func (LedgerAccount) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("sub_accounts", LedgerSubAccount.Type),
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
		// Mandatory Generic Dimensions
		field.String("currency_dimension_id").SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}).Immutable(),
		// Optional and Type Specific Dimensions
		field.String("tax_code_dimension_id").SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}).Optional().Immutable(),
		field.String("features_dimension_id").SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}).Optional().Immutable(),
		field.String("credit_priority_dimension_id").SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}).Optional().Immutable(),
	}
}

func (LedgerSubAccount) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "account_id", "currency_dimension_id").Unique(),
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
		edge.To("entries", LedgerEntry.Type),
		edge.From("currency_dimension", LedgerDimension.Type).
			Ref("currency_sub_accounts").
			Field("currency_dimension_id").
			Required().
			Immutable().
			Unique(),
		edge.From("tax_code_dimension", LedgerDimension.Type).
			Ref("tax_code_sub_accounts").
			Field("tax_code_dimension_id").
			Immutable().
			Unique(),
		edge.From("features_dimension", LedgerDimension.Type).
			Ref("features_sub_accounts").
			Field("features_dimension_id").
			Immutable().
			Unique(),
		edge.From("credit_priority_dimension", LedgerDimension.Type).
			Ref("credit_priority_sub_accounts").
			Field("credit_priority_dimension_id").
			Immutable().
			Unique(),
	}
}
