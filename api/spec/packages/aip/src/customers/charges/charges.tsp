import "../../shared/index.tsp";
import "../../productcatalog/index.tsp";
import "../../subscriptions/index.tsp";
import "../customer.tsp";

namespace Customers;

/**
 * Type of a charge.
 *
 * Values:
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
 * This is the same as `ProductCatalog.Discounts` but without the `usage` field, which is not applicable to flat fee charges.
 */
@friendlyName("BillingFlatFeeDiscounts")
@summary("Flat fee charge discounts")
model FlatFeeDiscounts {
  ...OmitProperties<ProductCatalog.Discounts, "usage">;
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
  @visibility(Lifecycle.Read)
  @summary("Type")
  type: T;

  /**
   * The customer owning the charge.
   */
  @visibility(Lifecycle.Read)
  @summary("Customer")
  customer: CustomerReference;

  /**
   * The charge is managed by the following entity.
   */
  @visibility(Lifecycle.Read)
  @summary("Managed by")
  managed_by: Shared.ResourceManagedBy;

  /**
   * The subscription that originated the charge, when the charge was created from a subscription item.
   */
  @visibility(Lifecycle.Read)
  @summary("Subscription")
  subscription?: Subscriptions.SubscriptionReference;

  /**
   * The currency of the charge.
   */
  @visibility(Lifecycle.Read)
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
  @visibility(Lifecycle.Read)
  @summary("Invoice at")
  invoice_at: Shared.DateTime;

  /**
   * The effective service period covered by the charge.
   */
  @visibility(Lifecycle.Read)
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
   * The earliest time when the charge should be advanced again by background processing.
   */
  @visibility(Lifecycle.Read)
  @summary("Advance after")
  advance_after?: Shared.DateTime;

  /**
   * The price of the charge.
   */
  @visibility(Lifecycle.Read)
  @summary("Price")
  price: Shared.CurrencyAmount;

  /**
   * Unique reference ID of the charge.
   */
  @visibility(Lifecycle.Read)
  @summary("Unique reference ID")
  unique_reference_id?: string;

  /**
   * Settlement mode of the charge.
   */
  @visibility(Lifecycle.Read)
  @summary("Settlement mode")
  settlement_mode: ProductCatalog.SettlementMode;
}

/**
 * A flat fee charge for a customer.
 */
@friendlyName("BillingFlatFeeCharge")
@summary("Flat fee charge")
model FlatFeeCharge {
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
  discounts?: FlatFeeDiscounts;

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

  // Calculated fields
  /**
   * The amount after proration of the charge.
   */
  @visibility(Lifecycle.Read)
  @summary("Amount after proration")
  amount_after_proration: Shared.CurrencyAmount;
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
  booked: Billing.BillingTotals;

  /**
   * The realtime amount of the charge.
   *
   * Requires the `realtime_usage` expand.
   */
  @visibility(Lifecycle.Read)
  @summary("Realtime totals")
  realtime?: Billing.BillingTotals;
}

/**
 * A usage-based charge for a customer.
 */
@friendlyName("BillingUsageBasedCharge")
@summary("Usage-based charge")
model UsageBasedCharge {
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

  /** Aggregated booked and realtime totals for the charge. */
  @visibility(Lifecycle.Read)
  @summary("Totals for the charge")
  totals: ChargeTotals;
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
  flat_fee: FlatFeeCharge,

  /**
   * A usage-based charge.
   */
  usage_based: UsageBasedCharge,
}

/**
 * Expands for customer charges.
 *
 * Values:
 * - `real_time_usage`: The charge's real-time usage.
 */
@friendlyName("BillingChargesExpand")
@summary("Customer charge expands")
enum ChargesExpand {
  RealTimeUsage: "real_time_usage",
}
