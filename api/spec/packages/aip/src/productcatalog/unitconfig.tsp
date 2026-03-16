import "../shared/index.tsp";
import "./price.tsp";

namespace ProductCatalog;

/**
 * The arithmetic operation used to convert raw metered units into billing units.
 */
@friendlyName("BillingUnitConfigOperation")
enum UnitConfigOperation {
  /** Divide the metered quantity by the conversion factor (e.g., bytes ÷ 1e9 = GB). */
  Divide: "divide",

  /** Multiply the metered quantity by the conversion factor (e.g., cost × 1.2 = cost + 20% margin). */
  Multiply: "multiply",
}

/**
 * The rounding mode applied to the converted quantity for invoicing.
 *
 * Rounding is applied only to the invoiced quantity.
 * Entitlement balance checks use the precise decimal value after conversion.
 */
@friendlyName("BillingUnitConfigRoundingMode")
enum UnitConfigRoundingMode {
  /** Round up to the next integer (typical for package-style billing). */
  Ceiling: "ceiling",

  /** Round down to the previous integer. */
  Floor: "floor",

  /** Round to the nearest integer, with 0.5 rounding up. */
  HalfUp: "half_up",

  /** No rounding; the converted value is used as-is. */
  None: "none",
}

/**
 * Unit conversion configuration.
 *
 * Transforms raw metered quantities into billing-ready units before pricing
 * and entitlement evaluation. Applied at the rate card level so the same
 * feature can be billed in different units across plans.
 *
 * Examples:
 * - Meter bytes, bill GB:        operation=divide, conversionFactor=1e9, rounding=ceiling, displayUnit="GB"
 * - Meter seconds, bill hours:   operation=divide, conversionFactor=3600, rounding=ceiling, displayUnit="hours"
 * - Cost + 20% margin:           operation=multiply, conversionFactor=1.2
 * - Bill per million tokens:     operation=divide, conversionFactor=1e6, rounding=ceiling, displayUnit="M"
 *
 * v1 equivalents:
 * - DynamicPrice(multiplier):              operation=multiply, conversionFactor=multiplier + UnitPrice(amount=1)
 * - PackagePrice(amount, quantityPerPkg):  operation=divide, conversionFactor=quantityPerPkg, rounding=ceiling + UnitPrice(amount)
 */
@friendlyName("BillingUnitConfig")
model UnitConfig {
  /**
   * The arithmetic operation to apply to the raw metered quantity.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Conversion operation")
  operation: UnitConfigOperation;

  /**
   * The factor used in the conversion operation.
   *
   * For divide: converted = raw / conversionFactor
   * For multiply: converted = raw × conversionFactor
   *
   * Must be a positive non-zero value.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Conversion factor")
  conversion_factor: Shared.Numeric;

  /**
   * The rounding mode applied to the converted quantity for invoicing.
   *
   * Defaults to none (no rounding).
   * Entitlement checks always use the precise (unrounded) value.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Rounding mode")
  rounding?: UnitConfigRoundingMode = UnitConfigRoundingMode.None;

  /**
   * The number of decimal places to retain after rounding.
   *
   * Only meaningful when rounding is not "none".
   * Defaults to 0 (round to whole numbers).
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Rounding precision (decimal places)")
  precision?: integer = 0;

  /**
   * A human-readable label for the converted unit shown on invoices and
   * in the customer portal (e.g., "GB", "hours", "M tokens").
   *
   * Optional. When omitted, no unit label is rendered.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Display unit label")
  display_unit?: string;
}

/**
 * Metered entitlement configuration for a usage-based rate card.
 *
 * When UnitConfig is present on the rate card, quota is expressed in
 * converted units (e.g., 100 means 100 GB if UnitConfig divides by 1e9).
 *
 * Entitlement balance checks use the precise (unrounded) converted value.
 */
@friendlyName("BillingMeteredEntitlement")
model MeteredEntitlement {
  /**
   * The usage quota in billing units (converted via UnitConfig when present).
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Quota")
  quota: Shared.Numeric;

  /**
   * The period over which the quota is enforced, as an ISO 8601 duration.
   *
   * Defaults to the billing cadence when omitted.
   * Can be shorter than the billing cadence for rate-limiting use cases
   * (e.g., "PT1H" for 1,000 tokens per hour on a monthly billing cycle).
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Quota enforcement period (ISO 8601 duration)")
  quota_period?: Shared.ISO8601Duration;

  /**
   * Whether usage beyond the quota is allowed (soft limit) or blocked (hard limit).
   *
   * - false (default): hard limit – access is denied when quota is exceeded.
   * - true: soft limit – overage is allowed and billed at the rate card's price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Allow overage (soft limit)")
  is_soft_limit?: boolean = false;
}

/**
 * Usage quantity details on an invoice line item when UnitConfig is in effect.
 *
 * Provides the full audit trail from raw meter output to the invoiced amount.
 */
@friendlyName("BillingInvoiceUsageQuantityDetail")
model InvoiceUsageQuantityDetail {
  /**
   * The raw quantity as reported by the meter (native units).
   */
  @visibility(Lifecycle.Read)
  @summary("Raw metered quantity")
  raw_quantity: Shared.Numeric;

  /**
   * The precise decimal value after applying the conversion operation and factor,
   * before rounding.
   */
  @visibility(Lifecycle.Read)
  @summary("Converted quantity (precise)")
  converted_quantity: Shared.Numeric;

  /**
   * The quantity after rounding, used for pricing.
   */
  @visibility(Lifecycle.Read)
  @summary("Invoiced quantity (rounded)")
  invoiced_quantity: Shared.Numeric;

  /**
   * The display unit label (e.g., "GB", "hours", "M tokens").
   */
  @visibility(Lifecycle.Read)
  @summary("Display unit")
  display_unit?: string;

  /**
   * Snapshot of the UnitConfig that was in effect at billing time.
   * Ensures historical invoices reflect the config that was actually applied.
   */
  @visibility(Lifecycle.Read)
  @summary("Applied unit config snapshot")
  applied_unit_config: UnitConfig;
}
