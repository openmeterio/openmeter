package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type LedgerBreakageRecord struct {
	ent.Schema
}

func (LedgerBreakageRecord) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.AnnotationsMixin{},
		entutils.TimeMixin{},
	}
}

func (LedgerBreakageRecord) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("kind").
			GoType(ledger.BreakageKind("")).
			Immutable(),
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
		field.Int("credit_priority").
			Immutable(),
		field.Time("expires_at").
			Immutable(),
		field.Enum("source_kind").
			GoType(ledger.BreakageSourceKind("")).
			Immutable(),
		field.String("source_transaction_group_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			Nillable().
			Immutable(),
		field.String("source_transaction_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			Nillable().
			Immutable(),
		field.String("breakage_transaction_group_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			NotEmpty().
			Immutable(),
		field.String("breakage_transaction_id").
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
		field.String("breakage_sub_account_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			NotEmpty().
			Immutable(),
		field.String("plan_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			Nillable().
			Immutable(),
		field.String("release_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			Nillable().
			Immutable(),
	}
}

func (LedgerBreakageRecord) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "customer_id", "currency", "credit_priority", "expires_at", "id"),
		index.Fields("namespace", "plan_id"),
		index.Fields("namespace", "source_transaction_group_id"),
		index.Fields("namespace", "breakage_transaction_group_id"),
	}
}
