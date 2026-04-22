package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/chargemeta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type ChargeFlatFee struct {
	ent.Schema
}

func (ChargeFlatFee) Mixin() []ent.Mixin {
	return []ent.Mixin{
		chargemeta.Mixin{},
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
			GoType(flatfee.ProRatingModeAdapterEnum("")),

		field.String("feature_key").
			Optional().
			NotEmpty().
			Nillable(),

		field.String("feature_id").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),

		field.Other("amount_before_proration", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),

		field.Other("amount_after_proration", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),

		field.Enum("status_detailed").
			GoType(flatfee.Status("")),
	}
}

func (ChargeFlatFee) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("credit_allocations", ChargeFlatFeeCreditAllocations.Type).
			StorageKey(edge.Symbol("charge_ff_credit_alloc_flat_fee")).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("detailed_lines", ChargeFlatFeeDetailedLine.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("invoiced_usage", ChargeFlatFeeInvoicedUsage.Type).
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("payment", ChargeFlatFeePayment.Type).
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("charge", Charge.Type).
			Unique().
			Immutable().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("subscription", Subscription.Type).
			Ref("charges_flat_fee").
			Field("subscription_id").
			Immutable().
			Unique(),
		edge.From("subscription_phase", SubscriptionPhase.Type).
			Ref("charges_flat_fee").
			Field("subscription_phase_id").
			Immutable().
			Unique(),
		edge.From("subscription_item", SubscriptionItem.Type).
			Ref("charges_flat_fee").
			Field("subscription_item_id").
			Immutable().
			Unique(),
		edge.From("customer", Customer.Type).
			Ref("charges_flat_fee").
			Field("customer_id").
			Unique().
			Required().
			Immutable(),
		edge.From("feature", Feature.Type).
			Ref("flat_fee_charges").
			Field("feature_id").
			Unique(),
	}
}

type ChargeFlatFeeDetailedLine struct {
	ent.Schema
}

func (ChargeFlatFeeDetailedLine) Mixin() []ent.Mixin {
	return []ent.Mixin{
		stddetailedline.Mixin{},
	}
}

func (ChargeFlatFeeDetailedLine) Fields() []ent.Field {
	return []ent.Field{
		field.String("charge_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
	}
}

func (ChargeFlatFeeDetailedLine) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("charge", ChargeFlatFee.Type).
			Ref("detailed_lines").
			Field("charge_id").
			Unique().
			Required(),
		edge.From("tax_code", TaxCode.Type).
			Ref("charge_flat_fee_detailed_lines").
			Field("tax_code_id").
			Unique(),
	}
}

func (ChargeFlatFeeDetailedLine) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "charge_id"),
		index.Fields("namespace", "charge_id", "child_unique_reference_id").
			Annotations(
				entsql.IndexWhere("deleted_at IS NULL"),
			).
			StorageKey("chargeffdetailedline_ns_charge_child_id").
			Unique(),
	}
}

func (ChargeFlatFeeDetailedLine) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "charge_flat_fee_detailed_line"},
	}
}

type ChargeFlatFeePayment struct {
	ent.Schema
}

func (ChargeFlatFeePayment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		payment.InvoicedMixin{},
	}
}

func (ChargeFlatFeePayment) Fields() []ent.Field {
	return []ent.Field{
		field.String("charge_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),
	}
}

func (ChargeFlatFeePayment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("billing_invoice_line", BillingInvoiceLine.Type).
			Ref("charge_flat_fee_payment").
			Field("line_id").
			Required().
			Immutable().
			Unique(),
		edge.From("flat_fee", ChargeFlatFee.Type).
			Ref("payment").
			Field("charge_id").
			Unique().
			Required().
			Immutable(),
	}
}

func (ChargeFlatFeePayment) Indexes() []ent.Index {
	return nil
}

type ChargeFlatFeeInvoicedUsage struct {
	ent.Schema
}

func (ChargeFlatFeeInvoicedUsage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		invoicedusage.Mixin{},
	}
}

func (ChargeFlatFeeInvoicedUsage) Fields() []ent.Field {
	return []ent.Field{
		field.String("charge_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),
	}
}

func (ChargeFlatFeeInvoicedUsage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("billing_invoice_line", BillingInvoiceLine.Type).
			Ref("charge_flat_fee_invoiced_usage").
			Field("line_id").
			Unique(),
		edge.From("flat_fee", ChargeFlatFee.Type).
			Ref("invoiced_usage").
			Field("charge_id").
			Unique().
			Required().
			Immutable(),
	}
}

func (ChargeFlatFeeInvoicedUsage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "charge_id").
			Unique(),
	}
}

type ChargeFlatFeeCreditAllocations struct {
	ent.Schema
}

func (ChargeFlatFeeCreditAllocations) Mixin() []ent.Mixin {
	return []ent.Mixin{
		creditrealization.Mixin{
			SelfReferenceType: ChargeFlatFeeCreditAllocations.Type,
		},
	}
}

func (ChargeFlatFeeCreditAllocations) Fields() []ent.Field {
	return []ent.Field{
		field.String("charge_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),
	}
}

func (ChargeFlatFeeCreditAllocations) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("flat_fee", ChargeFlatFee.Type).
			Ref("credit_allocations").
			Field("charge_id").
			Unique().
			Required().
			Immutable(),
		edge.From("billing_invoice_line", BillingInvoiceLine.Type).
			Ref("charge_flat_fee_credit_allocations").
			Field("line_id").
			Unique(),
	}
}
