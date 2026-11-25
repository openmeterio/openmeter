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

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	chargedetailedline "github.com/openmeterio/openmeter/openmeter/billing/charges/models/detailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/creditsapplied"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ChargeUsageBased struct {
	ent.Schema
}

func (ChargeUsageBased) Mixin() []ent.Mixin {
	return []ent.Mixin{
		ChargesMetaMixin{},
		ChargeValidationIssuesMixin{},
	}
}

func (ChargeUsageBased) Fields() []ent.Field {
	return []ent.Field{
		field.Time("invoice_at"),

		field.Enum("settlement_mode").
			GoType(productcatalog.SettlementMode("")).
			Immutable(),

		field.Time("intent_deleted_at").
			Optional().
			Nillable(),

		field.String("discounts").
			GoType(&billing.Discounts{}).
			ValueScanner(BillingDiscountsValueScanner).
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
			}).
			Optional().
			Nillable(),

		field.Enum("rating_engine").
			GoType(usagebased.RatingEngine("")),

		field.String("price").
			GoType(&productcatalog.Price{}).
			ValueScanner(PriceValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),

		// unit_config is a mutable rating input snapshotted from the effective rate
		// card, living in the mutable-fields layer alongside price and discounts
		// (price is mutable here, so the conversion that feeds it follows). Set on
		// create and update; cleared when absent.
		field.String("unit_config").
			GoType(&productcatalog.UnitConfig{}).
			ValueScanner(UnitConfigValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),

		field.String("current_realization_run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			Nillable(),

		field.String("cost_basis_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			Nillable().
			Immutable(),

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
		edge.To("revenue_recognitions", ChargeUsageBasedRevenueRecognitions.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("current_run", ChargeUsageBasedRuns.Type).
			Field("current_realization_run_id").
			Unique(),
		edge.To("cost_basis", ChargeUsageBasedCostBasis.Type).
			Field("cost_basis_id").
			StorageKey(edge.Symbol("charge_usage_based_cost_basis_charge_fk")).
			Unique().
			Immutable().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("charge", Charge.Type).
			Unique().
			Immutable().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("intent_override", ChargeUsageBasedOverride.Type).
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("subscription", Subscription.Type).
			Ref("charges_usage_based").
			Field("subscription_id").
			Unique().
			Immutable(),
		edge.From("subscription_phase", SubscriptionPhase.Type).
			Ref("charges_usage_based").
			Field("subscription_phase_id").
			Unique(),
		edge.From("subscription_item", SubscriptionItem.Type).
			Ref("charges_usage_based").
			Field("subscription_item_id").
			Unique(),
		edge.From("customer", Customer.Type).
			Field("customer_id").
			Ref("charges_usage_based").
			Unique().
			Required().
			Immutable(),
		edge.From("feature", Feature.Type).
			Field("feature_id").
			Ref("usage_based_charges").
			Unique(),
		edge.From("tax_code", TaxCode.Type).
			Ref("charge_usage_based").
			Field("tax_code_id").
			Unique().
			Required().
			Immutable().
			// We must not falsify tax code IDs on charges, when deleting a tax code (they have soft delete either ways).
			Annotations(entsql.OnDelete(entsql.Restrict)),
		edge.From("custom_currency", CustomCurrency.Type).
			Ref("charges_usage_based").
			Field("custom_currency_id").
			Unique().
			Immutable().
			Annotations(entsql.OnDelete(entsql.Restrict)),
	}
}

func (ChargeUsageBased) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tax_code_id").
			StorageKey("chargeusagebased_tax_code_id"),
		index.Fields("cost_basis_id").
			StorageKey("chargeusagebased_cost_basis_id").
			Unique(),
	}
}

func (ChargeUsageBased) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "charge_usage_based"},
	}
}

type ChargeUsageBasedCostBasis struct {
	ent.Schema
}

func (ChargeUsageBasedCostBasis) Mixin() []ent.Mixin {
	return []ent.Mixin{
		ChargeCostBasisMixin{},
	}
}

func (ChargeUsageBasedCostBasis) Edges() []ent.Edge {
	return chargeCostBasisCurrencyEdges("charge_usage_based_cost_basis")
}

func (ChargeUsageBasedCostBasis) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "charge_usage_based_cost_bases"},
	}
}

type ChargeUsageBasedOverride struct {
	ent.Schema
}

func (ChargeUsageBasedOverride) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
	}
}

func (ChargeUsageBasedOverride) Fields() []ent.Field {
	return []ent.Field{
		// The row has its own Ent ID, while charge_id is the semantic one-to-one
		// FK that makes override presence mean "this charge has an override".
		field.String("charge_id").
			Immutable().
			Unique().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),

		field.String("name").
			NotEmpty(),
		field.String("description").
			Optional().
			Nillable(),
		field.String("metadata").
			GoType(&models.Metadata{}).
			ValueScanner(entutils.JSONStringValueScanner[*models.Metadata]()).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),

		field.String("tax_behavior").
			GoType(productcatalog.TaxBehavior("")).
			Optional().
			Nillable().
			Deprecated("tax config overrides are not supported; use the base charge intent"),
		field.String("tax_code_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			Nillable().
			Deprecated("tax config overrides are not supported; use the base charge intent"),

		field.Time("intent_deleted_at").
			Optional().
			Nillable(),

		field.Time("service_period_from"),
		field.Time("service_period_to"),
		field.Time("full_service_period_from"),
		field.Time("full_service_period_to"),
		field.Time("billing_period_from"),
		field.Time("billing_period_to"),
		field.Time("invoice_at"),

		field.String("feature_key").
			Optional().
			NotEmpty().
			Nillable().
			Deprecated("feature key overrides are not supported; use the base usage-based charge intent"),
		field.String("price").
			GoType(&productcatalog.Price{}).
			ValueScanner(PriceValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
		field.String("discounts").
			GoType(&billing.Discounts{}).
			ValueScanner(BillingDiscountsValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
		field.String("unit_config").
			GoType(&productcatalog.UnitConfig{}).
			ValueScanner(UnitConfigValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),
	}
}

func (ChargeUsageBasedOverride) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tax_code_id").
			StorageKey("chargeusagebasedoverrides_tax_code_id"),
		index.Fields("namespace", "charge_id").Unique(),
	}
}

func (ChargeUsageBasedOverride) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("usage_based", ChargeUsageBased.Type).
			Ref("intent_override").
			Field("charge_id").
			Unique().
			Required().
			Immutable(),
		edge.From("tax_code", TaxCode.Type).
			Ref("charge_usage_based_overrides").
			Field("tax_code_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Restrict)),
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
			GoType(usagebased.RealizationRunType("")),

		field.Enum("initial_type").
			GoType(usagebased.RealizationRunType("")).
			Immutable(),

		field.Time("stored_at_lt"),

		field.Time("service_period_to").
			Immutable(),

		field.Int("schema_level").
			Default(usagebased.CurrentRealizationRunSchemaLevel).
			SchemaType(map[string]string{
				dialect.Postgres: "smallint",
			}),

		field.String("prior_run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			Nillable().
			Immutable(),

		field.Bool("detailed_lines_present"),

		field.Bool("detailed_lines_include_credit_allocations").
			Default(false),

		field.String("line_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			NotEmpty().
			Nillable(),

		field.String("invoice_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			NotEmpty().
			Nillable().
			Immutable(),

		field.Other("metered_quantity", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),

		field.Bool("no_fiat_transaction_required"),

		field.Bool("immutable").
			Default(false),

		field.Bool("fiat_overage_credit_allocation_completed").
			Default(false),
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
		edge.From("billing_invoice", BillingInvoice.Type).
			Ref("charge_usage_based_runs").
			Field("invoice_id").
			Unique().
			Immutable().
			Annotations(entsql.OnDelete(entsql.SetNull)),
		edge.To("next_runs", ChargeUsageBasedRuns.Type).
			StorageKey(edge.Symbol("charge_ub_run_prior_run")),
		edge.From("prior_run", ChargeUsageBasedRuns.Type).
			Ref("next_runs").
			Field("prior_run_id").
			Unique().
			Immutable(),
		edge.To("credit_allocations", ChargeUsageBasedRunCreditAllocations.Type).
			StorageKey(edge.Symbol("charge_ub_run_credit_alloc_run")).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("fiat_overage_credit_allocations", ChargeUsageBasedRunOverageCreditAllocations.Type).
			StorageKey(edge.Symbol("charge_ub_overage_credit_alloc_run")).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("detailed_lines", ChargeUsageBasedRunDetailedLine.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("corrected_detailed_lines", ChargeUsageBasedRunDetailedLine.Type).
			StorageKey(edge.Symbol("cub_run_corrected_detailed_lines")).
			Annotations(entsql.OnDelete(entsql.SetNull)),
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

type ChargeUsageBasedRevenueRecognitions struct {
	ent.Schema
}

func (ChargeUsageBasedRevenueRecognitions) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
		entutils.TimeMixin{},
		totals.Mixin{},
	}
}

func (ChargeUsageBasedRevenueRecognitions) Fields() []ent.Field {
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
			Immutable(),

		field.Time("stored_at_lt"),

		field.Time("service_period_to").
			Immutable(),

		field.String("prior_rev_recog_run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			Nillable().
			Immutable(),

		field.Other("metered_quantity", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),

		field.Bool("detailed_lines_include_credit_allocations"),
	}
}

func (ChargeUsageBasedRevenueRecognitions) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("usage_based", ChargeUsageBased.Type).
			Ref("revenue_recognitions").
			Field("charge_id").
			Unique().
			Required().
			Immutable(),
		edge.From("feature", Feature.Type).
			Field("feature_id").
			Ref("usage_based_revenue_recognitions").
			Unique().
			Required().
			Immutable(),
		edge.To("next_rev_recog_runs", ChargeUsageBasedRevenueRecognitions.Type).
			StorageKey(edge.Symbol("charge_ub_revrec_prior_rev_recog_run")),
		edge.From("prior_rev_recog_run", ChargeUsageBasedRevenueRecognitions.Type).
			Ref("next_rev_recog_runs").
			Field("prior_rev_recog_run_id").
			Unique().
			Immutable(),
		edge.To("credit_allocations", ChargeUsageBasedRevenueRecognitionCredits.Type).
			StorageKey(edge.Symbol("charge_ub_revrec_credits_run")).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("invoiced_usages", ChargeUsageBasedRevenueRecognitionInvoicedUsage.Type).
			StorageKey(edge.Symbol("charge_ub_revrec_invoiced_usage_run")).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("detailed_lines", ChargeUsageBasedRevenueRecognitionDetailedLine.Type).
			StorageKey(edge.Symbol("charge_ub_revrec_detailed_line_run")).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (ChargeUsageBasedRevenueRecognitions) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "charge_id"),
	}
}

type ChargeUsageBasedRevenueRecognitionCredits struct {
	ent.Schema
}

func (ChargeUsageBasedRevenueRecognitionCredits) Mixin() []ent.Mixin {
	return []ent.Mixin{
		creditrealization.Mixin{},
	}
}

func (ChargeUsageBasedRevenueRecognitionCredits) Fields() []ent.Field {
	return []ent.Field{
		field.String("run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),
	}
}

func (ChargeUsageBasedRevenueRecognitionCredits) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("run", ChargeUsageBasedRevenueRecognitions.Type).
			Ref("credit_allocations").
			Field("run_id").
			Unique().
			Required().
			Immutable(),
	}
}

func (ChargeUsageBasedRevenueRecognitionCredits) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "run_id").
			StorageKey("chargeubrevreccredits_ns_run"),
	}
}

// ChargeUsageBasedRevenueRecognitionInvoicedUsage holds the part of a revenue
// recognition run that credits do not cover. It has many rows per run so a
// partial revert can be stored as its own row.
type ChargeUsageBasedRevenueRecognitionInvoicedUsage struct {
	ent.Schema
}

func (ChargeUsageBasedRevenueRecognitionInvoicedUsage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		creditrealization.Mixin{},
	}
}

func (ChargeUsageBasedRevenueRecognitionInvoicedUsage) Fields() []ent.Field {
	return []ent.Field{
		field.String("run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),
	}
}

func (ChargeUsageBasedRevenueRecognitionInvoicedUsage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("run", ChargeUsageBasedRevenueRecognitions.Type).
			Ref("invoiced_usages").
			Field("run_id").
			Unique().
			Required().
			Immutable(),
	}
}

func (ChargeUsageBasedRevenueRecognitionInvoicedUsage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "run_id").
			StorageKey("chargeubrevrecinvoicedusage_ns_run"),
	}
}

type ChargeUsageBasedRevenueRecognitionDetailedLine struct {
	ent.Schema
}

// The columns are deliberately narrower than chargedetailedline.Mixin: one
// amount instead of totals, and no category, payment term, or invoicing app
// reference.
func (ChargeUsageBasedRevenueRecognitionDetailedLine) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.ResourceMixin{},
		entutils.AnnotationsMixin{},
	}
}

func (ChargeUsageBasedRevenueRecognitionDetailedLine) Fields() []ent.Field {
	return []ent.Field{
		field.String("run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),

		field.String("currency").
			GoType(currencyx.Code("")).
			NotEmpty().
			Immutable(),

		field.Time("service_period_start"),
		field.Time("service_period_end"),

		field.String("child_unique_reference_id").
			NotEmpty(),

		field.Int("index").
			Optional().
			Nillable(),

		field.Other("quantity", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),

		field.Other("per_unit_amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),

		field.Other("amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),

		field.JSON("amount_discounts", chargedetailedline.AmountDiscounts{}).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional(),

		field.JSON("credits_applied", &creditsapplied.CreditsApplied{}).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional(),
	}
}

func (ChargeUsageBasedRevenueRecognitionDetailedLine) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("run", ChargeUsageBasedRevenueRecognitions.Type).
			Ref("detailed_lines").
			Field("run_id").
			Unique().
			Required().
			Immutable(),
	}
}

func (ChargeUsageBasedRevenueRecognitionDetailedLine) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "run_id").
			StorageKey("chargeubrevrecdetailedline_ns_run"),
	}
}

func (ChargeUsageBasedRevenueRecognitionDetailedLine) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "charge_usage_based_revenue_recognition_detailed_line"},
	}
}

type ChargeUsageBasedRunDetailedLine struct {
	ent.Schema
}

func (ChargeUsageBasedRunDetailedLine) Mixin() []ent.Mixin {
	return []ent.Mixin{
		chargedetailedline.Mixin{},
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

		field.String("pricer_reference_id").
			NotEmpty(),
		field.String("corrects_run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			NotEmpty().
			Nillable(),
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
		edge.From("corrects_run", ChargeUsageBasedRuns.Type).
			Ref("corrected_detailed_lines").
			Field("corrects_run_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.SetNull)),
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
		creditrealization.Mixin{},
		creditrealization.CorrectionMixin{
			SelfReferenceType:   ChargeUsageBasedRunCreditAllocations.Type,
			SelfReferenceSymbol: "charge_usage_based_run_credit_allocations_charge_usage_based_ru",
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

type ChargeUsageBasedRunOverageCreditAllocations struct {
	ent.Schema
}

func (ChargeUsageBasedRunOverageCreditAllocations) Mixin() []ent.Mixin {
	return []ent.Mixin{
		creditrealization.Mixin{},
		creditrealization.CorrectionMixin{
			SelfReferenceType:   ChargeUsageBasedRunOverageCreditAllocations.Type,
			SelfReferenceSymbol: "charge_ub_overage_credit_alloc_correction",
		},
	}
}

func (ChargeUsageBasedRunOverageCreditAllocations) Fields() []ent.Field {
	return []ent.Field{
		field.String("run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),
	}
}

func (ChargeUsageBasedRunOverageCreditAllocations) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("run", ChargeUsageBasedRuns.Type).
			Ref("fiat_overage_credit_allocations").
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
