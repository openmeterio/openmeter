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
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/chargemeta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var CreditPurchaseSettlementValueScanner = entutils.JSONStringValueScanner[creditpurchase.Settlement]()

type ChargeCreditPurchase struct {
	ent.Schema
}

func (ChargeCreditPurchase) Mixin() []ent.Mixin {
	return []ent.Mixin{
		chargemeta.Mixin{},
	}
}

func (ChargeCreditPurchase) Fields() []ent.Field {
	return []ent.Field{
		// Intent fields
		field.Other("credit_amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Time("effective_at").
			Optional().
			Nillable().
			Immutable(),
		field.Int("priority").
			Optional().
			Nillable().
			Immutable(),

		field.String("settlement").
			GoType(creditpurchase.Settlement{}).
			ValueScanner(CreditPurchaseSettlementValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),

		field.Enum("status_detailed").
			GoType(creditpurchase.Status("")),
	}
}

func (ChargeCreditPurchase) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("external_payment", ChargeCreditPurchaseExternalPayment.Type).
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("invoiced_payment", ChargeCreditPurchaseInvoicedPayment.Type).
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("credit_grant", ChargeCreditPurchaseCreditGrant.Type).
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("charge", Charge.Type).
			Unique().
			Immutable().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("subscription", Subscription.Type).
			Ref("charges_credit_purchase").
			Field("subscription_id").
			Immutable().
			Unique(),
		edge.From("subscription_phase", SubscriptionPhase.Type).
			Ref("charges_credit_purchase").
			Field("subscription_phase_id").
			Immutable().
			Unique(),
		edge.From("subscription_item", SubscriptionItem.Type).
			Ref("charges_credit_purchase").
			Field("subscription_item_id").
			Immutable().
			Unique(),
		edge.From("customer", Customer.Type).
			Field("customer_id").
			Ref("charges_credit_purchase").
			Unique().
			Required().
			Immutable(),
	}
}

type ChargeCreditPurchaseCreditGrant struct {
	ent.Schema
}

func (ChargeCreditPurchaseCreditGrant) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
		entutils.TimeMixin{},
	}
}

func (ChargeCreditPurchaseCreditGrant) Fields() []ent.Field {
	return []ent.Field{
		field.String("charge_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),

		field.String("transaction_group_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			NotEmpty(),

		field.Time("granted_at"),
	}
}

func (ChargeCreditPurchaseCreditGrant) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("credit_purchase", ChargeCreditPurchase.Type).
			Ref("credit_grant").
			Field("charge_id").
			Unique().
			Required().
			Immutable(),
	}
}

func (ChargeCreditPurchaseCreditGrant) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "charge_id").
			Unique(),
	}
}

type ChargeCreditPurchaseExternalPayment struct {
	ent.Schema
}

func (ChargeCreditPurchaseExternalPayment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		payment.ExternalMixin{},
	}
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
	return nil
}

type ChargeCreditPurchaseInvoicedPayment struct {
	ent.Schema
}

func (ChargeCreditPurchaseInvoicedPayment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		payment.InvoicedMixin{},
	}
}

func (ChargeCreditPurchaseInvoicedPayment) Fields() []ent.Field {
	return []ent.Field{
		field.String("charge_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),
	}
}

func (ChargeCreditPurchaseInvoicedPayment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("billing_invoice_line", BillingInvoiceLine.Type).
			Ref("charge_credit_purchase_invoiced_payment").
			Field("line_id").
			Required().
			Immutable().
			Unique(),
		edge.From("credit_purchase", ChargeCreditPurchase.Type).
			Ref("invoiced_payment").
			Field("charge_id").
			Unique().
			Required().
			Immutable(),
	}
}

func (ChargeCreditPurchaseInvoicedPayment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "charge_id").
			Unique(),
	}
}
