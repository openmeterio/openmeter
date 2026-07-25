import "../../shared/index.tsp";
import "../../productcatalog/index.tsp";
import "../../subscriptions/index.tsp";
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
   */
  @visibility(Lifecycle.Read)
  @summary("Customer")
  customer: CustomerReference;

  /**
   * Indicates whether the charge lifecycle is controlled by OpenMeter or manually
   * overridden by the API user.
   */
  @visibility(Lifecycle.Read)
  @summary("Lifecycle controller")
  lifecycle_controller: Shared.LifecycleController;

  /**
   * The subscription that originated the charge, when the charge was created from a
   * subscription item.
   */
  @visibility(Lifecycle.Read)
  @summary("Subscription")
  subscription?: Subscriptions.SubscriptionReference;

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
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Feature key")
  feature_key?: string;

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
 * The totals of a change.
 *
 * RealTime is only expanded when the `real_time_usage` expand is used.
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
   * The realtime amount of the charge.
   *
   * Requires the `realtime_usage` expand.
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
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Feature key")
  feature_key: string;

  // Status fields

  /**
   * Aggregated booked and realtime totals for the charge.
   */
  @visibility(Lifecycle.Read)
  @summary("Totals for the charge")
  totals: ChargeTotals;

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
 * - `real_time_usage`: The charge's real-time usage.
 */
@friendlyName("BillingChargesExpand")
@summary("Customer charge expands")
enum ChargesExpand {
  RealTimeUsage: "real_time_usage",
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
  ...OmitProperties<ChargeFlatFee, "full_service_period" | "billing_period">;

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
  ...OmitProperties<ChargeUsageBased, "full_service_period" | "billing_period">;

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
