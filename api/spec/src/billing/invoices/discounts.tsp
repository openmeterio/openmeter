import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

namespace OpenMeter;

/**
 * InvoiceDiscountBase represents a charge or discount that can be applied to a line or the entire invoice.
 */
@friendlyName("InvoiceDiscountBase")
model InvoiceDiscountBase {
  ...ResourceTimestamps;

  /**
   * ID of the charge or discount.
   */
  @visibility(Lifecycle.Read)
  id: ULID;

  /**
   * Reason code.
   */
  @visibility(Lifecycle.Read)
  reason: BillingDiscountReason;

  /**
   * Text description as to why the discount was applied.
   */
  @visibility(Lifecycle.Read)
  description?: string;

  /**
   * External IDs of the invoice in other apps such as Stripe.
   */
  @visibility(Lifecycle.Read)
  externalIds?: InvoiceLineAppExternalIds;
}

/**
 * InvoiceLineDiscounts represents the discounts applied to the invoice line by type.
 */
@friendlyName("InvoiceLineDiscounts")
model InvoiceLineDiscounts {
  /**
   * Amount based discounts applied to the line.
   *
   * Amount based discounts are deduced from the total price of the line.
   */
  amount?: InvoiceLineAmountDiscount[];

  /**
   * Usage based discounts applied to the line.
   *
   * Usage based discounts are deduced from the usage of the line before price calculations are applied.
   */
  usage?: InvoiceLineUsageDiscount[];
}

/**
 * InvoiceLineUsageDiscount represents an usage-based discount applied to the line.
 *
 * The deduction is done before the pricing algorithm is applied.
 */
@friendlyName("InvoiceLineUsageDiscount")
model InvoiceLineUsageDiscount {
  ...InvoiceDiscountBase;

  /**
   * The usage to apply.
   */
  @summary("Usage quantity in the unit of the underlying meter")
  @visibility(Lifecycle.Read)
  quantity: Numeric;

  /**
   * The usage discount already applied to the previous split lines.
   *
   * Only set if progressive billing is enabled and the line is a split line.
   */
  @summary("Usage quantity in the unit of the underlying meter")
  @visibility(Lifecycle.Read)
  preLinePeriodQuantity?: Numeric;
}

/**
 * InvoiceLineAmountDiscount represents an amount deducted from the line, and will be applied before taxes.
 */
@friendlyName("InvoiceLineAmountDiscount")
model InvoiceLineAmountDiscount {
  ...InvoiceDiscountBase;

  /**
   * Fixed discount amount to apply (calculated if percent present).
   */
  @visibility(Lifecycle.Read)
  @summary("Amount in the currency of the invoice")
  amount: Money;
}

/**
 * Billing specific fields for product catalog discounts.
 */
@friendlyName("BillingDiscountMetadata")
model BillingDiscountMetadata {
  /**
   * Correlation ID for the discount.
   *
   * This is used to link discounts across different invoices (progressive billing use case).
   *
   * If not provided, the invoicing engine will auto-generate one. When editing an invoice line,
   * please make sure to keep the same correlation ID of the discount or in progressive billing
   * setups the discount amounts might be incorrect.
   */
  correlationId?: ULID;
}

/**
 * A percentage discount.
 */
@friendlyName("BillingDiscountPercentage")
model BillingDiscountPercentage {
  ...DiscountPercentage;
  ...BillingDiscountMetadata;
}

/**
 * A usage discount.
 */
@friendlyName("BillingDiscountUsage")
model BillingDiscountUsage {
  ...DiscountUsage;
  ...BillingDiscountMetadata;
}

/**
 * A discount by type.
 */
@friendlyName("BillingDiscounts")
model BillingDiscounts {
  /**
   * The percentage discount.
   */
  percentage?: BillingDiscountPercentage;

  /**
   * The usage discount.
   */
  usage?: BillingDiscountUsage;
}

/**
 * The type of the discount reason.
 */
@friendlyName("DiscountReasonType")
enum DiscountReasonType {
  maximumSpend: "maximum_spend",
  ratecardPercentage: "ratecard_percentage",
  ratecardUsage: "ratecard_usage",
}

/**
 * The reason for the discount.
 */
@friendlyName("BillingDiscountReason")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union BillingDiscountReason {
  maximum_spend: DiscountReasonMaximumSpend,
  ratecard_percentage: DiscountReasonRatecardPercentage,
  ratecard_usage: DiscountReasonRatecardUsage,
}

/**
 * The reason for the discount is a maximum spend.
 */
@friendlyName("DiscountReasonMaximumSpend")
model DiscountReasonMaximumSpend {
  type: DiscountReasonType.maximumSpend;
}

/**
 * The reason for the discount is a ratecard percentage.
 */
@friendlyName("DiscountReasonRatecardPercentage")
model DiscountReasonRatecardPercentage {
  type: DiscountReasonType.ratecardPercentage;
  ...BillingDiscountPercentage;
}

/**
 * The reason for the discount is a ratecard usage.
 */
@friendlyName("DiscountReasonRatecardUsage")
model DiscountReasonRatecardUsage {
  type: DiscountReasonType.ratecardUsage;
  ...BillingDiscountUsage;
}
