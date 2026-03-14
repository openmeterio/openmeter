package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var CreditPurchaseSettlementValueScanner = entutils.JSONStringValueScanner[creditpurchase.Settlement]()

type ChargeCreditPurchase struct {
	ent.Schema
}

func (ChargeCreditPurchase) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
	}
}

func (ChargeCreditPurchase) Fields() []ent.Field {
	return []ent.Field{
		// Intent fields
		field.Other("credit_amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),

		field.String("settlement").
			GoType(creditpurchase.Settlement{}).
			ValueScanner(CreditPurchaseSettlementValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),

		field.String("credit_grant_transaction_group_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			NotEmpty().
			Nillable(),

		field.Time("credit_granted_at").Optional().Nillable(),
	}
}

func (ChargeCreditPurchase) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("charge", Charge.Type).
			Ref("credit_purchase").
			Unique().
			Required(),
		edge.To("external_payment", ChargeCreditPurchaseExternalPayment.Type).
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

type ChargeCreditPurchaseExternalPayment struct {
	ent.Schema
}

func (ChargeCreditPurchaseExternalPayment) Fields() []ent.Field {
	return []ent.Field{
		field.String("charge_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),
	}
}

func (ChargeCreditPurchaseExternalPayment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		payment.ExternalMixin{},
	}
}

func (ChargeCreditPurchaseExternalPayment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("credit_purchase", ChargeCreditPurchase.Type).
			Ref("external_payment").
			Field("charge_id").
			Unique().
			Required().
			Immutable(),
	}
}

func (ChargeCreditPurchaseExternalPayment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "charge_id").
			Unique(),
	}
}
