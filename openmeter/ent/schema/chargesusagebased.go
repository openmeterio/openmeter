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

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/chargemeta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type ChargeUsageBased struct {
	ent.Schema
}

func (ChargeUsageBased) Mixin() []ent.Mixin {
	return []ent.Mixin{
		chargemeta.Mixin{},
	}
}

func (ChargeUsageBased) Fields() []ent.Field {
	return []ent.Field{
		field.Time("invoice_at").
			Immutable(),

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

		field.String("feature_key").
			NotEmpty().
			Immutable(),

		field.String("feature_id").
			NotEmpty().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),

		field.String("price").
			GoType(&productcatalog.Price{}).
			ValueScanner(PriceValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Immutable(),

		field.String("current_realization_run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			Nillable(),

		field.Enum("status_detailed").
			GoType(usagebased.Status("")),
	}
}

func (ChargeUsageBased) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("runs", ChargeUsageBasedRuns.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("detailed_lines", ChargeUsageBasedRunDetailedLine.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("current_run", ChargeUsageBasedRuns.Type).
			Field("current_realization_run_id").
			Unique(),
		edge.To("charge", Charge.Type).
			Unique().
			Immutable().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("subscription", Subscription.Type).
			Ref("charges_usage_based").
			Field("subscription_id").
			Unique().
			Immutable(),
		edge.From("subscription_phase", SubscriptionPhase.Type).
			Ref("charges_usage_based").
			Field("subscription_phase_id").
			Unique().
			Immutable(),
		edge.From("subscription_item", SubscriptionItem.Type).
			Ref("charges_usage_based").
			Field("subscription_item_id").
			Unique().
			Immutable(),
		edge.From("customer", Customer.Type).
			Field("customer_id").
			Ref("charges_usage_based").
			Unique().
			Required().
			Immutable(),
		edge.From("feature", Feature.Type).
			Field("feature_id").
			Ref("usage_based_charges").
			Unique().
			Required(),
	}
}

func (ChargeUsageBased) Indexes() []ent.Index {
	return nil
}

func (ChargeUsageBased) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "charge_usage_based"},
	}
}

type ChargeUsageBasedRuns struct {
	ent.Schema
}

func (ChargeUsageBasedRuns) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
		entutils.TimeMixin{},
		totals.Mixin{},
	}
}

func (ChargeUsageBasedRuns) Fields() []ent.Field {
	return []ent.Field{
		field.String("charge_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),

		field.String("feature_id").
			NotEmpty().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable().
			Comment("For future-proofing runs may diverge from the charge feature later, but today this matches the parent charge feature_id."),

		field.Enum("type").
			GoType(usagebased.RealizationRunType("")).
			Immutable(),

		field.Time("asof"),

		field.Time("collection_end").
			Immutable(),

		field.String("line_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			NotEmpty().
			Nillable(),

		field.Other("meter_value", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
	}
}

func (ChargeUsageBasedRuns) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("usage_based", ChargeUsageBased.Type).
			Ref("runs").
			Field("charge_id").
			Unique().
			Required().
			Immutable(),
		edge.From("feature", Feature.Type).
			Field("feature_id").
			Ref("usage_based_runs").
			Unique().
			Required().
			Immutable(),
		edge.From("billing_invoice_line", BillingInvoiceLine.Type).
			Ref("charge_usage_based_run").
			Field("line_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.SetNull)),
		edge.To("credit_allocations", ChargeUsageBasedRunCreditAllocations.Type).
			StorageKey(edge.Symbol("charge_ub_run_credit_alloc_run")).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("detailed_lines", ChargeUsageBasedRunDetailedLine.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("invoiced_usage", ChargeUsageBasedRunInvoicedUsage.Type).
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("payment", ChargeUsageBasedRunPayment.Type).
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (ChargeUsageBasedRuns) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "charge_id"),
	}
}

type ChargeUsageBasedRunDetailedLine struct {
	ent.Schema
}

func (ChargeUsageBasedRunDetailedLine) Mixin() []ent.Mixin {
	return []ent.Mixin{
		stddetailedline.Mixin{},
	}
}

func (ChargeUsageBasedRunDetailedLine) Fields() []ent.Field {
	return []ent.Field{
		field.String("charge_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),

		field.String("run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
	}
}

func (ChargeUsageBasedRunDetailedLine) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("charge", ChargeUsageBased.Type).
			Ref("detailed_lines").
			Field("charge_id").
			Unique().
			Required(),
		edge.From("run", ChargeUsageBasedRuns.Type).
			Ref("detailed_lines").
			Field("run_id").
			Unique().
			Required(),
		edge.From("tax_code", TaxCode.Type).
			Ref("charge_usage_based_run_detailed_lines").
			Field("tax_code_id").
			Unique(),
	}
}

func (ChargeUsageBasedRunDetailedLine) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "charge_id"),
		index.Fields("namespace", "run_id"),
		index.Fields("namespace", "charge_id", "run_id", "child_unique_reference_id").
			Annotations(
				entsql.IndexWhere("deleted_at IS NULL"),
			).
			StorageKey("chargeubdetailedline_ns_charge_run_child_id").
			Unique(),
	}
}

func (ChargeUsageBasedRunDetailedLine) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "charge_usage_based_run_detailed_line"},
	}
}

type ChargeUsageBasedRunCreditAllocations struct {
	ent.Schema
}

func (ChargeUsageBasedRunCreditAllocations) Mixin() []ent.Mixin {
	return []ent.Mixin{
		creditrealization.Mixin{
			SelfReferenceType: ChargeUsageBasedRunCreditAllocations.Type,
		},
	}
}

func (ChargeUsageBasedRunCreditAllocations) Fields() []ent.Field {
	return []ent.Field{
		field.String("run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),
	}
}

func (ChargeUsageBasedRunCreditAllocations) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("run", ChargeUsageBasedRuns.Type).
			Ref("credit_allocations").
			Field("run_id").
			Unique().
			Required().
			Immutable(),
	}
}

type ChargeUsageBasedRunInvoicedUsage struct {
	ent.Schema
}

func (ChargeUsageBasedRunInvoicedUsage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		invoicedusage.Mixin{},
	}
}

func (ChargeUsageBasedRunInvoicedUsage) Fields() []ent.Field {
	return []ent.Field{
		field.String("run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),
	}
}

func (ChargeUsageBasedRunInvoicedUsage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("run", ChargeUsageBasedRuns.Type).
			Ref("invoiced_usage").
			Field("run_id").
			Unique().
			Required().
			Immutable(),
	}
}

type ChargeUsageBasedRunPayment struct {
	ent.Schema
}

func (ChargeUsageBasedRunPayment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		payment.InvoicedMixin{},
	}
}

func (ChargeUsageBasedRunPayment) Fields() []ent.Field {
	return []ent.Field{
		field.String("run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),
	}
}

func (ChargeUsageBasedRunPayment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("run", ChargeUsageBasedRuns.Type).
			Ref("payment").
			Field("run_id").
			Unique().
			Required().
			Immutable(),
	}
}
