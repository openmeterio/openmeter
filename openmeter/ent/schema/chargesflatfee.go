package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
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
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
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

		field.String("current_realization_run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			Nillable(),

		field.Enum("status_detailed").
			GoType(flatfee.Status("")),
	}
}

func (ChargeFlatFee) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("credit_allocations", ChargeFlatFeeRunCreditAllocations.Type).
			StorageKey(edge.Symbol("charge_ff_credit_alloc_flat_fee")).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("detailed_lines", ChargeFlatFeeRunDetailedLine.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("invoiced_usage", ChargeFlatFeeRunInvoicedUsage.Type).
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("payment", ChargeFlatFeeRunPayment.Type).
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("runs", ChargeFlatFeeRun.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("current_run", ChargeFlatFeeRun.Type).
			Field("current_realization_run_id").
			Unique(),
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
		edge.From("tax_code", TaxCode.Type).
			Ref("charge_flat_fees").
			Field("tax_code_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.SetNull)),
	}
}

func (ChargeFlatFee) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tax_code_id").
			StorageKey("chargeflatfees_tax_code_id"),
	}
}

type ChargeFlatFeeRun struct {
	ent.Schema
}

func (ChargeFlatFeeRun) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
		entutils.TimeMixin{},
		totals.Mixin{},
	}
}

func (ChargeFlatFeeRun) Fields() []ent.Field {
	return []ent.Field{
		field.String("charge_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),

		field.Enum("type").
			GoType(flatfee.RealizationRunType("")),

		field.Enum("initial_type").
			GoType(flatfee.RealizationRunType("")).
			Immutable(),

		field.Time("service_period_from"),
		field.Time("service_period_to"),

		field.Other("amount_after_proration", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
	}
}

func (ChargeFlatFeeRun) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("flat_fee", ChargeFlatFee.Type).
			Ref("runs").
			Field("charge_id").
			Unique().
			Required().
			Immutable(),
		edge.To("credit_allocations", ChargeFlatFeeRunCreditAllocations.Type).
			StorageKey(edge.Symbol("charge_ff_credit_alloc_run")).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("detailed_lines", ChargeFlatFeeRunDetailedLine.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("invoiced_usage", ChargeFlatFeeRunInvoicedUsage.Type).
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("payment", ChargeFlatFeeRunPayment.Type).
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (ChargeFlatFeeRun) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "charge_id"),
	}
}

type ChargeFlatFeeRunDetailedLine struct {
	ent.Schema
}

func (ChargeFlatFeeRunDetailedLine) Mixin() []ent.Mixin {
	return []ent.Mixin{
		stddetailedline.Mixin{},
	}
}

func (ChargeFlatFeeRunDetailedLine) Fields() []ent.Field {
	return []ent.Field{
		field.String("charge_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			Nillable().
			Deprecated("flat-fee realization ownership is stored on run_id"),

		field.String("run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),

		field.String("pricer_reference_id").
			NotEmpty(),
	}
}

func (ChargeFlatFeeRunDetailedLine) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("charge", ChargeFlatFee.Type).
			Ref("detailed_lines").
			Field("charge_id").
			Unique(),
		edge.From("run", ChargeFlatFeeRun.Type).
			Ref("detailed_lines").
			Field("run_id").
			Unique().
			Required().
			Immutable(),
		edge.From("tax_code", TaxCode.Type).
			Ref("charge_flat_fee_run_detailed_lines").
			Field("tax_code_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.SetNull)),
	}
}

func (ChargeFlatFeeRunDetailedLine) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "charge_id"),
		index.Fields("namespace", "run_id"),
		index.Fields("namespace", "charge_id", "child_unique_reference_id").
			Annotations(
				entsql.IndexWhere("deleted_at IS NULL"),
			).
			StorageKey("chargeffrundetailedline_ns_charge_child_id").
			Unique(),
		index.Fields("namespace", "run_id", "child_unique_reference_id").
			Annotations(
				entsql.IndexWhere("deleted_at IS NULL"),
			).
			StorageKey("chargeffdetailedline_ns_run_child_id").
			Unique(),
	}
}

type ChargeFlatFeeRunPayment struct {
	ent.Schema
}

func (ChargeFlatFeeRunPayment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		payment.InvoicedMixin{},
	}
}

func (ChargeFlatFeeRunPayment) Fields() []ent.Field {
	return []ent.Field{
		field.String("charge_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			Nillable().
			Immutable().
			Deprecated("flat-fee payment ownership is stored on run_id"),
		field.String("run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),
	}
}

func (ChargeFlatFeeRunPayment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("billing_invoice_line", BillingInvoiceLine.Type).
			Ref("charge_flat_fee_run_payment").
			Field("line_id").
			Required().
			Immutable().
			Unique(),
		edge.From("flat_fee", ChargeFlatFee.Type).
			Ref("payment").
			Field("charge_id").
			Unique().
			Immutable(),
		edge.From("run", ChargeFlatFeeRun.Type).
			Ref("payment").
			Field("run_id").
			Unique().
			Required().
			Immutable(),
	}
}

func (ChargeFlatFeeRunPayment) Indexes() []ent.Index {
	return nil
}

type ChargeFlatFeeRunInvoicedUsage struct {
	ent.Schema
}

func (ChargeFlatFeeRunInvoicedUsage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		invoicedusage.Mixin{},
	}
}

func (ChargeFlatFeeRunInvoicedUsage) Fields() []ent.Field {
	return []ent.Field{
		field.String("charge_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			Nillable().
			Immutable().
			Deprecated("flat-fee invoiced usage ownership is stored on run_id"),
		field.String("run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),
	}
}

func (ChargeFlatFeeRunInvoicedUsage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("billing_invoice_line", BillingInvoiceLine.Type).
			Ref("charge_flat_fee_run_invoiced_usage").
			Field("line_id").
			Unique(),
		edge.From("flat_fee", ChargeFlatFee.Type).
			Ref("invoiced_usage").
			Field("charge_id").
			Unique().
			Immutable(),
		edge.From("run", ChargeFlatFeeRun.Type).
			Ref("invoiced_usage").
			Field("run_id").
			Unique().
			Required().
			Immutable(),
	}
}

func (ChargeFlatFeeRunInvoicedUsage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "charge_id").
			Unique(),
		index.Fields("namespace", "run_id").
			Unique(),
	}
}

type ChargeFlatFeeRunCreditAllocations struct {
	ent.Schema
}

func (ChargeFlatFeeRunCreditAllocations) Mixin() []ent.Mixin {
	return []ent.Mixin{
		creditrealization.Mixin{
			SelfReferenceType: ChargeFlatFeeRunCreditAllocations.Type,
		},
	}
}

func (ChargeFlatFeeRunCreditAllocations) Fields() []ent.Field {
	return []ent.Field{
		field.String("charge_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			Nillable().
			Immutable().
			Deprecated("flat-fee credit allocation ownership is stored on run_id"),
		field.String("run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),
	}
}

func (ChargeFlatFeeRunCreditAllocations) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("flat_fee", ChargeFlatFee.Type).
			Ref("credit_allocations").
			Field("charge_id").
			Unique().
			Immutable(),
		edge.From("run", ChargeFlatFeeRun.Type).
			Ref("credit_allocations").
			Field("run_id").
			Unique().
			Required().
			Immutable(),
		edge.From("billing_invoice_line", BillingInvoiceLine.Type).
			Ref("charge_flat_fee_run_credit_allocations").
			Field("line_id").
			Unique(),
	}
}
