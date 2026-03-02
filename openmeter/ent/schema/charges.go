package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Charge struct {
	ent.Schema
}

func (Charge) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.AnnotationsMixin{},
		entutils.ResourceMixin{},
	}
}

func (Charge) Fields() []ent.Field {
	return []ent.Field{
		field.String("customer_id").
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),

		field.Time("service_period_from"),
		field.Time("service_period_to"),
		field.Time("billing_period_from"),
		field.Time("billing_period_to"),
		field.Time("full_service_period_from"),
		field.Time("full_service_period_to"),

		field.Enum("type").
			GoType(charges.ChargeType("")).
			Immutable(),

		field.Enum("status").
			GoType(charges.ChargeStatus("")),

		field.String("unique_reference_id").
			Optional().
			Nillable(),

		field.String("currency").
			GoType(currencyx.Code("")).
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "varchar(3)",
			}),

		field.Enum("managed_by").
			GoType(billing.InvoiceLineManagedBy("")),

		// Subscriptions metadata
		field.String("subscription_id").
			Optional().
			Nillable().
			Immutable(),

		field.String("subscription_phase_id").
			Optional().
			Nillable().
			Immutable(),

		field.String("subscription_item_id").
			Optional().
			Nillable().
			Immutable(),
	}
}

func (Charge) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "customer_id", "unique_reference_id").
			Annotations(
				entsql.IndexWhere("unique_reference_id IS NOT NULL AND deleted_at IS NULL"),
			).
			Unique(),
	}
}

func (Charge) Edges() []ent.Edge {
	return []ent.Edge{
		// Union type edges
		edge.To("flat_fee", ChargeFlatFee.Type).
			StorageKey(edge.Column("id")).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		edge.To("usage_based", ChargeUsageBased.Type).
			StorageKey(edge.Column("id")).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		edge.To("credit_purchase", ChargeCreditPurchase.Type).
			StorageKey(edge.Column("id")).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		// Realizations
		edge.To("credit_realizations", ChargeCreditRealization.Type),
		// Billing
		edge.To("billing_invoice_lines", BillingInvoiceLine.Type),
		edge.To("billing_split_line_groups", BillingInvoiceSplitLineGroup.Type),
		// Customer
		edge.From("customer", Customer.Type).
			Ref("charge_intents").
			Field("customer_id").
			Unique().
			Immutable().
			Required(),
		// Subscriptions
		edge.From("subscription", Subscription.Type).
			Ref("charge_intents").
			Field("subscription_id").
			Unique().
			Immutable(),
		edge.From("subscription_phase", SubscriptionPhase.Type).
			Ref("charge_intents").
			Field("subscription_phase_id").
			Unique().
			Immutable(),
		edge.From("subscription_item", SubscriptionItem.Type).
			Ref("charge_intents").
			Field("subscription_item_id").
			Unique().
			Immutable(),
	}
}

type ChargeUsageBased struct {
	ent.Schema
}

func (ChargeUsageBased) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
	}
}

func (ChargeUsageBased) Fields() []ent.Field {
	return []ent.Field{
		field.String("price").
			GoType(&productcatalog.Price{}).
			ValueScanner(PriceValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),

		field.String("feature_key").
			NotEmpty(),

		field.Time("invoice_at"),

		field.Enum("settlement_mode").
			GoType(productcatalog.SettlementMode("")).
			Immutable(),

		field.String("discounts").
			GoType(&productcatalog.Discounts{}).
			ValueScanner(DiscountsValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),
	}
}

func (ChargeUsageBased) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id").
			Unique(),
	}
}

func (ChargeUsageBased) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("charge", Charge.Type).
			Ref("usage_based").
			Unique().
			Required(),
	}
}

type ChargeFlatFee struct {
	ent.Schema
}

func (ChargeFlatFee) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
	}
}

func (ChargeFlatFee) Fields() []ent.Field {
	return []ent.Field{
		field.String("payment_term").
			GoType(productcatalog.PaymentTermType("")).
			NotEmpty(),

		field.Time("invoice_at"),

		field.Enum("settlement_mode").
			GoType(productcatalog.SettlementMode("")).
			Immutable(),

		field.String("discounts").
			GoType(&productcatalog.Discounts{}).
			ValueScanner(DiscountsValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),

		field.Enum("pro_rating").
			GoType(charges.ProRatingModeAdapterEnum("")),

		field.String("feature_key").
			Optional().
			NotEmpty().
			Nillable(),

		field.Other("amount_before_proration", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),

		field.Other("amount_after_proration", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
	}
}

func (ChargeFlatFee) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("charge", Charge.Type).
			Ref("flat_fee").
			Unique().
			Required(),
		edge.To("charge_standard_invoice_payment_settlement", ChargeStandardInvoicePaymentSettlement.Type).
			Unique(),
		edge.To("charge_standard_invoice_accrued_usage", ChargeStandardInvoiceAccruedUsage.Type).
			Unique(),
	}
}

func (ChargeFlatFee) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id").
			Unique(),
	}
}

var CreditPurchaseSettlementValueScanner = entutils.JSONStringValueScanner[charges.CreditPurchaseSettlement]()

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
			GoType(charges.CreditPurchaseSettlement{}).
			ValueScanner(CreditPurchaseSettlementValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),

		// State fields

		field.Enum("status").
			GoType(charges.PaymentSettlementStatus("")),
	}
}

func (ChargeCreditPurchase) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("charge", Charge.Type).
			Ref("credit_purchase").
			Unique().
			Required(),
	}
}

type ChargeStandardInvoicePaymentSettlement struct {
	ent.Schema
}

func (ChargeStandardInvoicePaymentSettlement) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
		entutils.TimeMixin{},
		entutils.AnnotationsMixin{},
	}
}

func (ChargeStandardInvoicePaymentSettlement) Fields() []ent.Field {
	return []ent.Field{
		field.String("line_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),

		field.String("charge_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),

		field.Time("service_period_from"),
		field.Time("service_period_to"),

		field.Enum("status").
			GoType(charges.StandardInvoicePaymentSettlementStatus("")),

		field.Other("amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),

		// TODO: Let's add edges to ledger
		field.String("authorized_transaction_group_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			NotEmpty().
			Nillable(),

		field.Time("authorized_at").Optional().Nillable(),

		field.String("settled_transaction_group_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			NotEmpty().
			Nillable(),

		field.Time("settled_at").Optional().Nillable(),
	}
}

func (ChargeStandardInvoicePaymentSettlement) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "charge_id", "line_id").
			Annotations(
				entsql.IndexWhere("line_id IS NOT NULL AND deleted_at IS NULL"),
			).
			Unique(),
	}
}

func (ChargeStandardInvoicePaymentSettlement) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("billing_invoice_line", BillingInvoiceLine.Type).
			Ref("charge_standard_invoice_payment_settlement").
			Field("line_id").
			Unique().
			Required().
			Immutable(),
		edge.From("flat_fee", ChargeFlatFee.Type).
			Ref("charge_standard_invoice_payment_settlement").
			Field("charge_id").
			Unique().
			Required().
			Immutable(),
	}
}

type ChargeStandardInvoiceAccruedUsage struct {
	ent.Schema
}

func (ChargeStandardInvoiceAccruedUsage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
		entutils.TimeMixin{},
		entutils.AnnotationsMixin{},
		TotalsMixin{},
	}
}

func (ChargeStandardInvoiceAccruedUsage) Fields() []ent.Field {
	return []ent.Field{
		field.String("charge_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),

		field.String("line_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			NotEmpty().
			Nillable(),

		field.Time("service_period_from"),
		field.Time("service_period_to"),

		// Mutable flag indicates if the accrued usage can be reallocated as credits or if this needs to happen via
		// the invoicing flow.
		field.Bool("mutable"),
	}
}

func (ChargeStandardInvoiceAccruedUsage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("billing_invoice_line", BillingInvoiceLine.Type).
			Ref("charge_standard_invoice_accrued_usage").
			Field("line_id").
			Unique(),
		edge.From("flat_fee", ChargeFlatFee.Type).
			Ref("charge_standard_invoice_accrued_usage").
			Field("charge_id").
			Unique().
			Required().
			Immutable(),
	}
}

func (ChargeStandardInvoiceAccruedUsage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "charge_id", "line_id").
			Annotations(
				entsql.IndexWhere("line_id IS NOT NULL AND deleted_at IS NULL"),
			).
			Unique(),
	}
}

type ChargeCreditRealization struct {
	ent.Schema
}

func (ChargeCreditRealization) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
		entutils.TimeMixin{},
		entutils.AnnotationsMixin{},
	}
}

func (ChargeCreditRealization) Fields() []ent.Field {
	return []ent.Field{
		field.String("charge_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),

		field.String("line_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			NotEmpty().
			Nillable(),

		field.Other("amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),

		field.Time("service_period_from"),
		field.Time("service_period_to"),
	}
}

func (ChargeCreditRealization) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("charge", Charge.Type).
			Ref("credit_realizations").
			Field("charge_id").
			Unique().
			Required().
			Immutable(),
		edge.From("billing_invoice_line", BillingInvoiceLine.Type).
			Ref("charge_credit_realization").
			Field("line_id").
			Unique(),
	}
}
