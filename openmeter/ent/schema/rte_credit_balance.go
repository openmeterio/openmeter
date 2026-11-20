package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type RTECreditBalance struct {
	ent.Schema
}

func (RTECreditBalance) Mixin() []ent.Mixin {
	return []ent.Mixin{entutils.NamespaceMixin{}, entutils.TimeMixin{}}
}

func (RTECreditBalance) Fields() []ent.Field {
	return []ent.Field{
		field.String("customer_id").NotEmpty().Immutable().SchemaType(map[string]string{dialect.Postgres: "char(26)"}),
		field.String("currency_key").NotEmpty().Immutable(),
		field.String("feature_scope").NotEmpty().Immutable(),
		field.Time("calculated_at"),
		field.Other("booked", alpacadecimal.Decimal{}).SchemaType(map[string]string{dialect.Postgres: "numeric"}),
		field.Other("spendable", alpacadecimal.Decimal{}).SchemaType(map[string]string{dialect.Postgres: "numeric"}),
		field.Other("live", alpacadecimal.Decimal{}).SchemaType(map[string]string{dialect.Postgres: "numeric"}),
		field.Other("pending", alpacadecimal.Decimal{}).SchemaType(map[string]string{dialect.Postgres: "numeric"}),
	}
}

func (RTECreditBalance) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "customer_id", "currency_key", "feature_scope").Unique().
			StorageKey("rte_credit_balance_scope").Annotations(entsql.IndexWhere("deleted_at IS NULL")),
	}
}
