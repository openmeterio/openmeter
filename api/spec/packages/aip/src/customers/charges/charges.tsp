import "../../shared/index.tsp";
import "../../productcatalog/index.tsp";
import "../../subscriptions/index.tsp";
import "../../features/index.tsp";
import "../customer.tsp";

namespace Customers;

/**
 * Type of a charge.
 *
 * Values:
 *
 * - `flat_fee`: A fixed-amount charge.
 * - `usage_based`: A usage-priced charge.
 */
@friendlyName("BillingChargeType")
@summary("Charge type")
enum ChargeType {
  FlatFee: "flat_fee",
  UsageBased: "usage_based",
}

/**
 * Lifecycle status of a charge.
 *
 * Values:
 *
 * - `created`: The charge has been created but is not active yet.
 * - `active`: The charge is active.
 * - `final`: The charge is fully finalized and no further changes are expected.
 * - `deleted`: The charge has been deleted.
 */
@friendlyName("BillingChargeStatus")
@summary("Charge status")
enum ChargeStatus {
  Created: "created",
  Active: "active",
  Final: "final",
  Deleted: "deleted",
}

/**
 * Discounts applicable to flat fee charges.
 *
 * This is the same as `ProductCatalog.Discounts` but without the `usage` field,
 * which is not applicable to flat fee charges.
 */
@friendlyName("BillingChargeFlatFeeDiscounts")
@summary("Flat fee charge discounts")
model ChargeFlatFeeDiscounts {
  ...OmitProperties<ProductCatalog.Discounts, "usage">;
}

/**
 * Flat fee intent fields from the system lifecycle controller shadowed by a manual
 * override.
 */
@friendlyName("BillingChargeFlatFeeSystemIntent")
@summary("Flat fee system lifecycle controller charge intent")
model ChargeFlatFeeSystemIntent {
  ...PickProperties<
    ChargeFlatFee,
    | "name"
    | "description"
    | "labels"
    | "invoice_at"
    | "service_period"
    | "full_service_period"
    | "billing_period"
    | "payment_term"
    | "discounts"
    | "proration_configuration"
  >;

  /**
   * The amount before proration of the system lifecycle controller flat fee intent.
   */
  @visibility(Lifecycle.Read)
  @summary("Amount before proration")
  amount_before_proration: Shared.CurrencyAmount;

  /**
   * The timestamp when the system lifecycle controller intent was deleted. The
   * effective charge can remain visible while a manual override is active.
   */
  @visibility(Lifecycle.Read)
  @summary("Deleted at")
  deleted_at?: Shared.DateTime;
}

/**
 * Usage-based intent fields from the system lifecycle controller shadowed by a
 * manual override.
 */
@friendlyName("BillingChargeUsageBasedSystemIntent")
@summary("Usage-based system lifecycle controller charge intent")
model ChargeUsageBasedSystemIntent {
  ...PickProperties<
    ChargeUsageBased,
    | "name"
    | "description"
    | "labels"
    | "invoice_at"
    | "service_period"
    | "full_service_period"
    | "billing_period"
    | "discounts"
    | "price"
  >;

  /**
   * The timestamp when the system lifecycle controller intent was deleted. The
   * effective charge can remain visible while a manual override is active.
   */
  @visibility(Lifecycle.Read)
  @summary("Deleted at")
  deleted_at?: Shared.DateTime;
}

/**
 * Settlement status of a charge realization's payment.
 *
 * Values:
 *
 * - `authorized`: The payment has been authorized against the customer's ledger
 *   but has not settled yet.
 * - `settled`: The payment has settled.
 */
@friendlyName("BillingChargeRealizationPaymentStatus")
@summary("Charge realization payment status")
enum ChargeRealizationPaymentStatus {
  Authorized: "authorized",
  Settled: "settled",
}

/**
 * Payment state of a charge realization.
 */
@friendlyName("BillingChargeRealizationPayment")
@summary("Charge realization payment")
model ChargeRealizationPayment {
  /**
   * The settlement status of the payment.
   */
  @visibility(Lifecycle.Read)
  @summary("Status")
  status: ChargeRealizationPaymentStatus;
}

/**
 * Type of a charge realization run.
 *
 * Values:
 *
 * - `final_realization`: The run is the final realization of the charge for its
 *   service period; no further runs are expected to correct it.
 * - `partial_invoice`: The run only realizes part of the charge, because the
 *   remainder is not yet due to be invoiced.
 * - `outstanding`: The type an outstanding (not-yet-realized) projection would
 *   carry if it were realized now. Never returned on a booked realization run.
 */
@friendlyName("BillingChargeRealizationType")
@summary("Charge realization type")
enum ChargeRealizationType {
  FinalRealization: "final_realization",
  PartialInvoice: "partial_invoice",
  Outstanding: "outstanding",
}

/**
 * Cost category of a charge realization detailed line.
 *
 * Values:
 *
 * - `regular`: A regular charge line, based on usage or subscription pricing.
 * - `commitment`: A commitment-derived line, such as minimum spend.
 */
@friendlyName("BillingChargeRealizationDetailedLineCategory")
@summary("Charge realization detailed line category")
enum ChargeRealizationDetailedLineCategory {
  Regular: "regular",
  Commitment: "commitment",
}

/**
 * A credit allocation applied to a charge realization detailed line.
 */
@friendlyName("BillingChargeRealizationDetailedLineCreditApplied")
@summary("Charge realization detailed line credit applied")
model ChargeRealizationDetailedLineCreditApplied {
  /**
   * The monetary amount credited.
   */
  @visibility(Lifecycle.Read)
  @summary("Amount")
  amount: Shared.Numeric;

  /**
   * Human-readable description of the credit allocation.
   */
  @visibility(Lifecycle.Read)
  @summary("Description")
  description?: string;

  /**
   * The ID of the credit realization (allocation) this credit was applied from.
   */
  @visibility(Lifecycle.Read)
  @summary("Credit realization ID")
  credit_realization_id: string;
}

/**
 * Shared fields present on every charge realization detailed line, regardless
 * of charge type.
 *
 * Deliberately independent of an invoice's own detailed lines: a realization
 * detailed line is the charge run's own rated breakdown, which exists even
 * when no invoice is ever produced (e.g. a credits-only settlement charge).
 * Its shape must not be coupled to the invoice line's, which only exists once
 * a realization has actually been invoiced and can evolve independently.
 */
@friendlyName("BillingChargeRealizationDetailedLineBase")
@summary("Charge realization detailed line")
model ChargeRealizationDetailedLineBase<T extends ChargeType> {
  ...OmitProperties<Shared.Resource, "name" | "description" | "labels">;

  /**
   * The type of the charge the realization belongs to.
   */
  @visibility(Lifecycle.Read)
  @summary("Type")
  type: T;

  /**
   * The service period covered by this detailed line.
   */
  @visibility(Lifecycle.Read)
  @summary("Service period")
  service_period: Shared.ClosedPeriod;

  /**
   * Aggregated financial totals for the detailed line.
   */
  @visibility(Lifecycle.Read)
  @summary("Totals")
  totals: Common.BillingRoot.BillingTotals;

  /**
   * The cost category of this detailed line.
   */
  @visibility(Lifecycle.Read)
  @summary("Category")
  category: ChargeRealizationDetailedLineCategory;

  /**
   * Credits applied to this detailed line.
   */
  @visibility(Lifecycle.Read)
  @summary("Credits applied")
  credits_applied?: ChargeRealizationDetailedLineCreditApplied[];

  /**
   * The unit price of the detailed line.
   */
  @visibility(Lifecycle.Read)
  @summary("Unit price")
  unit_price: Shared.Numeric;

  /**
   * The amount discounts applied to the detailed line.
   */
  @visibility(Lifecycle.Read)
  @summary("Amount discounts")
  amount_discounts: ChargeRealizationAmountDiscount[];
}

/**
 * A discount applied to a charge realization detailed line.
 */
@friendlyName("BillingChargeRealizationAmountDiscount")
@summary("Charge detailed line amount discount")
model ChargeRealizationAmountDiscount {
  /**
   * The amount of the discount applied to the charge.
   */
  @visibility(Lifecycle.Read)
  @summary("Amount")
  amount: Shared.Numeric;

  /**
   * The reason for the discount applied to the charge.
   */
  @visibility(Lifecycle.Read)
  @summary("Reason")
  reason: string;

  /**
   * Optional human-readable description of the discount.
   */
  @visibility(Lifecycle.Read)
  @summary("Description")
  description?: string;

  /**
   * The discount amount applied to the charge, rounded to the nearest cent (or
   * currency equivalent) and adjusted to compensate for rounding errors across
   * multiple discounted detail lines.
   */
  @visibility(Lifecycle.Read)
  @summary("Rounded amount")
  rounding_amount?: Shared.Numeric;

  /**
   * Unique reference ID of the discount applied to the charge.
   */
  @visibility(Lifecycle.Read)
  @summary("Unique reference ID")
  child_unique_reference_id: string;
}

/**
 * A detailed (child) line type of a charge realization run.
 *
 * This is distinct from an invoice's own detailed lines: it represents the
 * rated/priced breakdown produced by the realization run itself, before that
 * breakdown is (or is not yet) reflected on an invoice line. Credit-then-invoice
 * runs include credit allocations in these lines, while credits-only runs keep the
 * gross rated detail.
 */
@friendlyName("BillingChargeRealizationDetailedLine")
@summary("Charge realization detailed line")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union ChargeRealizationDetailedLine {
  flat_fee: ChargeRealizationDetailedLineFlatFee,
  usage_based: ChargeRealizationDetailedLineUsageBased,
}

/**
 * A detailed line produced by a flat fee charge's realization run.
 */
@friendlyName("BillingChargeRealizationDetailedLineFlatFee")
@summary("Flat fee charge realization detailed line")
model ChargeRealizationDetailedLineFlatFee {
  ...ChargeRealizationDetailedLineBase<ChargeType.FlatFee>;
}

/**
 * A detailed line produced by a usage-based charge's realization run.
 */
@friendlyName("BillingChargeRealizationDetailedLineUsageBased")
@summary("Usage-based charge realization detailed line")
model ChargeRealizationDetailedLineUsageBased {
  ...ChargeRealizationDetailedLineBase<ChargeType.UsageBased>;

  /**
   * The quantity of the detailed line.
   */
  @visibility(Lifecycle.Read)
  @summary("Quantity")
  quantity: Shared.Numeric;

  /**
   * The ID of a prior realization run this detailed line corrects. Only set if the
   * detailed line's totals are negative.
   */
  @visibility(Lifecycle.Read)
  @summary("Corrects run ID")
  corrects_run_id?: string;
}

/**
 * A realization run of a charge.
 *
 * `totals` and `detailed_lines` are only populated with the `realization.totals`
 * and `realization.detailed_lines` expands, respectively, since computing them
 * requires re-deriving the run's rated breakdown. `invoice` is an ID reference
 * unless the `realization.invoice` expand is used, which resolves it to the full
 * invoice.
 */
@friendlyName("BillingChargeRealization")
@summary("Charge realization")
model ChargeRealization {
  /**
   * The ID of the realization run. Not present on `outstanding` entries, which are
   * projections rather than persisted runs.
   */
  @visibility(Lifecycle.Read)
  @summary("ID")
  id?: string;

  /**
   * The ID of the invoice line this realization was booked to, when the realization
   * has been invoiced.
   */
  @visibility(Lifecycle.Read)
  @summary("Line ID")
  line_id?: string;

  /**
   * The reference of the invoice related to the realization.
   */
  @visibility(Lifecycle.Read)
  @summary("Invoice")
  invoice?: Shared.ExpandableResourceReference<Invoices.Invoice>;

  /**
   * The type of the realization run.
   */
  @visibility(Lifecycle.Read)
  @summary("Type")
  type: ChargeRealizationType;

  /**
   * The service period covered by this realization run.
   */
  @visibility(Lifecycle.Read)
  @summary("Service period")
  service_period: Shared.ClosedPeriod;

  /**
   * The metered usage quantity this realization run accounts for.
   */
  @visibility(Lifecycle.Read)
  @summary("Usage")
  usage: Shared.Numeric;

  /**
   * The payment state of the realization, when the charge requires a fiat
   * transaction to settle.
   */
  @visibility(Lifecycle.Read)
  @summary("Payment")
  payment?: ChargeRealizationPayment;

  /**
   * Financial totals for the realization run, including credit allocations.
   *
   * Requires the `realization.totals` expand.
   */
  @visibility(Lifecycle.Read)
  @summary("Totals")
  totals?: Common.BillingRoot.BillingTotals;

  /**
   * The detailed (rated) lines produced by the realization run.
   *
   * Requires the `realization.detailed_lines` expand.
   */
  @visibility(Lifecycle.Read)
  @summary("Detailed lines")
  detailed_lines?: ChargeRealizationDetailedLine[];
}

/**
 * Shared fields present on all customer charge responses.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "charge temporal fields should remain flat"
@friendlyName("BillingChargeBase")
@summary("Charge")
model ChargeBase<T extends ChargeType> {
  ...Shared.Resource;

  /**
   * The type of the charge.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Type")
  type: T;

  /**
   * The customer owning the charge.
   *
   * By default, only the `id` of the customer is returned. For more details use the
   * `customer` expand.
   */
  @visibility(Lifecycle.Read)
  @summary("Customer")
  customer: Shared.ExpandableResourceReference<Customer>;

  /**
   * Indicates whether the charge lifecycle is controlled by the system or manually
   * overridden by the API user.
   */
  @visibility(Lifecycle.Read)
  @summary("Lifecycle controller")
  lifecycle_controller: Shared.LifecycleController;

  /**
   * The subscription that originated the charge, when the charge was created from a
   * subscription item.
   *
   * By default, only the `id`, `phase.id`, `phase.item.id` of the subscription is
   * returned. For more details use the `subscription` expand.
   */
  @visibility(Lifecycle.Read)
  @summary("Subscription")
  subscription?: Shared.CustomExpandableResourceReference<
    Subscriptions.Subscription,
    Subscriptions.SubscriptionReference
  >;

  /**
   * The currency of the charge.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Currency")
  currency: Shared.CurrencyCode;

  /**
   * The lifecycle status of the charge.
   */
  @visibility(Lifecycle.Read)
  @summary("Status")
  status: ChargeStatus;

  /**
   * The timestamp when the charge is intended to be invoiced.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Invoice at")
  invoice_at: Shared.DateTime;

  /**
   * The effective service period covered by the charge.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Service period")
  service_period: Shared.ClosedPeriod;

  /**
   * The full, unprorated service period of the charge.
   */
  @visibility(Lifecycle.Read)
  @summary("Full service period")
  full_service_period: Shared.ClosedPeriod;

  /**
   * The billing period the charge belongs to.
   */
  @visibility(Lifecycle.Read)
  @summary("Billing period")
  billing_period: Shared.ClosedPeriod;

  /**
   * The earliest time when the charge should be advanced again by background
   * processing.
   */
  @visibility(Lifecycle.Read)
  @summary("Advance after")
  advance_after?: Shared.DateTime;

  /**
   * Unique reference ID of the charge.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Unique reference ID")
  unique_reference_id?: string;

  /**
   * Settlement mode of the charge.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Settlement mode")
  settlement_mode: ProductCatalog.SettlementMode;

  /**
   * Tax configuration of the charge.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Tax configuration")
  tax_config?: Common.BillingRoot.TaxConfig;

  /**
   * The realization runs of the charge, sorted by `service_period.from`.
   */
  @visibility(Lifecycle.Read)
  @summary("Realizations")
  realizations: ChargeRealization[];
}

/**
 * A flat fee charge for a customer.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" ""
@friendlyName("BillingChargeFlatFee")
@summary("Flat fee charge")
model ChargeFlatFee {
  ...ChargeBase<ChargeType.FlatFee>;

  /**
   * Payment term of the flat fee charge.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Payment term")
  payment_term: ProductCatalog.PricePaymentTerm;

  /**
   * The discounts applied to the charge.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Discounts")
  discounts?: ChargeFlatFeeDiscounts;

  /**
   * The feature associated with the charge, when applicable.
   */
  @visibility(Lifecycle.Read)
  @summary("Feature")
  feature?: Shared.ExpandableResourceReference<Features.Feature>;

  /**
   * The proration configuration of the charge.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Proration configuration")
  proration_configuration: ProductCatalog.ProrationConfiguration;

  /**
   * The amount before proration of the charge.
   */
  @visibility(Lifecycle.Create)
  @summary("Amount before proration")
  amount_before_proration: Shared.CurrencyAmount;

  // Calculated fields
  /**
   * The amount after proration of the charge.
   */
  @visibility(Lifecycle.Read)
  @summary("Amount after proration")
  amount_after_proration: Shared.CurrencyAmount;

  /**
   * The price of the charge.
   */
  @visibility(Lifecycle.Read)
  @summary("Price")
  price: ProductCatalog.Price;

  /**
   * Current intent from the system lifecycle controller for a charge that has an
   * active manual override. The top-level charge fields remain the effective
   * customer-facing intent.
   */
  @visibility(Lifecycle.Read)
  @summary("System intent")
  system_intent?: ChargeFlatFeeSystemIntent;
}

/**
 * The totals of a charge.
 *
 * `realtime` is only populated when the `real_time_usage` expand is used.
 */
@friendlyName("BillingChargeTotals")
model ChargeTotals {
  /**
   * The amount of the charge already booked to the internal accounting system.
   */
  @visibility(Lifecycle.Read)
  @summary("Booked")
  booked: Common.BillingRoot.BillingTotals;

  /**
   * The realtime amount of the charge, i.e. the whole usage rated at the charge's
   * price for its full service period, ignoring what has already been booked to a
   * realization. This differs from the `totals` of a realization with type
   * `outstanding`, which only cover the portion not yet booked.
   *
   * Requires the `real_time_usage` expand.
   */
  @visibility(Lifecycle.Read)
  @summary("Realtime totals")
  realtime?: Common.BillingRoot.BillingTotals;
}

/**
 * A usage-based charge for a customer.
 */
@friendlyName("BillingChargeUsageBased")
@summary("Usage-based charge")
model ChargeUsageBased {
  ...ChargeBase<ChargeType.UsageBased>;

  /**
   * Discounts applied to the usage-based charge.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Discounts")
  discounts?: ProductCatalog.Discounts;

  /**
   * The feature associated with the charge.
   */
  @visibility(Lifecycle.Read)
  @summary("Feature")
  feature: Shared.ExpandableResourceReference<Features.Feature>;

  /**
   * Aggregated booked and realtime totals for the charge.
   */
  @visibility(Lifecycle.Read)
  @summary("Totals for the charge")
  totals: ChargeTotals;

  /**
   * The metered usage quantity of the charge for its full service period.
   *
   * Requires the `real_time_usage` expand, since it is computed live from the
   * metering store rather than stored on the charge.
   */
  @visibility(Lifecycle.Read)
  @summary("Usage")
  usage?: Shared.Numeric;

  /**
   * The price of the charge.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Price")
  price: ProductCatalog.Price;

  /**
   * Current intent from the system lifecycle controller for a charge that has an
   * active manual override. The top-level charge fields remain the effective
   * customer-facing intent.
   */
  @visibility(Lifecycle.Read)
  @summary("System intent")
  system_intent?: ChargeUsageBasedSystemIntent;
}

/**
 * Customer charge.
 */
@friendlyName("BillingCharge")
@summary("Customer charge")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union Charge {
  /**
   * A flat fee charge.
   */
  flat_fee: ChargeFlatFee,

  /**
   * A usage-based charge.
   */
  usage_based: ChargeUsageBased,
}

/**
 * Expands for customer charges.
 *
 * Values:
 *
 * - `real_time_usage`: The charge's real-time usage; it sets the `usage` and the
 *   `totals.realtime` fields.
 * - `customer`: The complete customer entity of the charge.
 * - `feature`: The complete feature entity of the charge.
 * - `subscription`: The complete subscription of the charge, when present.
 * - `realization.invoice`: The complete invoice entity of each realization, in
 *   place of the ID reference.
 * - `realization.totals`: The `totals` of each realization run, including credit
 *   allocations.
 * - `realization.detailed_lines`: The `detailed_lines` of each realization run.
 */
@friendlyName("BillingChargesExpand")
@summary("Customer charge expands")
enum ChargesExpand {
  RealTimeUsage: "real_time_usage",
  Customer: "customer",
  Feature: "feature",
  Subscription: "subscription",
  RealizationInvoice: "realization.invoice",
  RealizationTotals: "realization.totals",
  RealizationDetailedLines: "realization.detailed_lines",
}

/**
 * Customer charge.
 */
@friendlyName("CreateChargeRequest")
@summary("Customer charge create request.")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union CreateCharge {
  /**
   * A flat fee charge.
   */
  flat_fee: CreateChargeFlatFee,

  /**
   * A usage-based charge.
   */
  usage_based: CreateChargeUsageBased,
}

/**
 * Create a flat fee charge for a customer.
 */
@doc("Flat fee charge create request.")
@withVisibility(Lifecycle.Create)
@friendlyName("CreateChargeFlatFeeRequest")
model CreateChargeFlatFee {
  ...OmitProperties<
    ChargeFlatFee,
    "full_service_period" | "billing_period" | "feature"
  >;

  /**
   * A reference to the feature associated with the charge, when applicable.
   */
  @visibility(Lifecycle.Create)
  @summary("Feature")
  feature?: Shared.ResourceReference<Features.Feature>;

  /**
   * The full, unprorated service period of the charge.
   */
  @visibility(Lifecycle.Create)
  @summary("Full service period")
  full_service_period?: Shared.ClosedPeriod;

  /**
   * The billing period the charge belongs to.
   */
  @visibility(Lifecycle.Create)
  @summary("Billing period")
  billing_period?: Shared.ClosedPeriod;
}

/**
 * Create a usage-based charge for a customer.
 */
@doc("Usage-based charge create request.")
@withVisibility(Lifecycle.Create)
@friendlyName("CreateChargeUsageBasedRequest")
model CreateChargeUsageBased {
  ...OmitProperties<
    ChargeUsageBased,
    "full_service_period" | "billing_period" | "feature"
  >;

  /**
   * A reference to the feature associated with the charge.
   */
  @visibility(Lifecycle.Create)
  @summary("Feature")
  feature: Shared.ResourceReference<Features.Feature>;

  /**
   * The full, unprorated service period of the charge.
   */
  @visibility(Lifecycle.Create)
  @summary("Full service period")
  full_service_period?: Shared.ClosedPeriod;

  /**
   * The billing period the charge belongs to.
   */
  @visibility(Lifecycle.Create)
  @summary("Billing period")
  billing_period?: Shared.ClosedPeriod;
}

/**
 * Reference to a charge associated with an invoice line.
 */
@friendlyName("BillingChargeReference")
model ChargeReference {
  /**
   * Unique identifier for the charge.
   */
  @visibility(Lifecycle.Read)
  @summary("ID")
  id: Shared.ULID;
}
