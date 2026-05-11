import "../shared/index.tsp";
import "./price.tsp";

namespace ProductCatalog;

/**
 * The arithmetic operation used to convert raw metered units into billing units.
 *
 * - `divide`: Divide the metered quantity by the conversion factor (e.g., bytes ÷
 *   1e9 = GB).
 * - `multiply`: Multiply the metered quantity by the conversion factor (e.g., cost
 *   × 1.2 = cost + 20% margin).
 */
@friendlyName("BillingUnitConfigOperation")
enum UnitConfigOperation {
  Divide: "divide",
  Multiply: "multiply",
}

/**
 * The rounding mode applied to the converted quantity for invoicing.
 *
 * Rounding is applied only to the invoiced quantity. Entitlement balance checks
 * use the precise decimal value after conversion.
 *
 * - `ceiling`: Round up to the next integer (typical for package-style billing).
 * - `floor`: Round down to the previous integer.
 * - `half_up`: Round to the nearest integer, with 0.5 rounding up.
 * - `none`: No rounding; the converted value is used as-is.
 */
@friendlyName("BillingUnitConfigRoundingMode")
enum UnitConfigRoundingMode {
  Ceiling: "ceiling",
  Floor: "floor",
  HalfUp: "half_up",
  None: "none",
}

/**
 * Unit conversion configuration.
 *
 * Transforms raw metered quantities into billing-ready units before pricing and
 * entitlement evaluation. Applied at the rate card level so the same feature can
 * be billed in different units across plans.
 *
 * Examples:
 *
 * - Meter bytes, bill GB: operation=divide, conversionFactor=1e9,
 *   rounding=ceiling, displayUnit="GB"
 * - Meter seconds, bill hours: operation=divide, conversionFactor=3600,
 *   rounding=ceiling, displayUnit="hours"
 * - Cost + 20% margin: operation=multiply, conversionFactor=1.2
 * - Bill per million tokens: operation=divide, conversionFactor=1e6,
 *   rounding=ceiling, displayUnit="M"
 *
 * v1 equivalents:
 *
 * - DynamicPrice(multiplier): operation=multiply, conversionFactor=multiplier +
 *   UnitPrice(amount=1)
 * - PackagePrice(amount, quantityPerPkg): operation=divide,
 *   conversionFactor=quantityPerPkg, rounding=ceiling + UnitPrice(amount)
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
   * - For `divide`: `converted = raw / conversionFactor`.
   * - For `multiply`: `converted = raw × conversionFactor`.
   *
   * Must be a positive non-zero value.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Conversion factor")
  conversion_factor: Shared.Numeric;

  /**
   * The rounding mode applied to the converted quantity for invoicing.
   *
   * Defaults to none (no rounding). Entitlement checks always use the precise
   * (unrounded) value.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Rounding mode")
  rounding?: UnitConfigRoundingMode = UnitConfigRoundingMode.None;

  /**
   * The number of decimal places to retain after rounding.
   *
   * Only meaningful when rounding is not "none". Defaults to 0 (round to whole
   * numbers).
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Rounding precision (decimal places)")
  precision?: integer = 0;

  /**
   * A human-readable label for the converted unit shown on invoices and in the
   * customer portal (e.g., "GB", "hours", "M tokens").
   *
   * Optional. When omitted, no unit label is rendered.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Display unit label")
  display_unit?: string;
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
   * Snapshot of the UnitConfig that was in effect at billing time. Ensures
   * historical invoices reflect the config that was actually applied.
   */
  @visibility(Lifecycle.Read)
  @summary("Applied unit config snapshot")
  applied_unit_config: UnitConfig;
}
