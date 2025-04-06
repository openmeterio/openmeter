import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

namespace Invoices;

/**
 * Reason code.
 */
@friendlyName("InvoiceDiscountReason")
enum InvoiceDiscountReason {
  maximumSpend: "maximum_spend",
  ratecardDiscount: "ratecard_discount",
}

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
  reason: InvoiceDiscountReason;

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

  /**
   * The discount from the rate card this discount is based on.
   */
  @visibility(Lifecycle.Read)
  rateCardDiscount?: OpenMeter.ProductCatalog.Discount;
}

/**
 * The type of the discount on the invoice line.
 */
@friendlyName("InvoiceLineDiscountType")
enum InvoiceLineDiscountType {
  amount: "amount",
  usage: "usage",
}

/**
 * InvoiceLineDiscount represents the actual discount applied to the invoice line.
 */
@friendlyName("InvoiceLineDiscount")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union InvoiceLineDiscount {
  amount: InvoiceLineDiscountAmount,
  usage: InvoiceLineDiscountUsage,
}

/**
 * InvoiceLineDiscountUsage represents an usage-based discount applied to the line.
 *
 * The deduction is done before the pricing algorithm is applied.
 */
@friendlyName("InvoiceLineDiscountUsage")
model InvoiceLineDiscountUsage {
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

  /**
   * The type of the discount.
   */
  @visibility(Lifecycle.Read)
  type: InvoiceLineDiscountType.usage;
}

/**
 * InvoiceLineDiscountAmount represents an amount deducted from the line, and will be applied before taxes.
 */
@friendlyName("InvoiceLineDiscountAmount")
model InvoiceLineDiscountAmount {
  ...InvoiceDiscountBase;

  /**
   * Fixed discount amount to apply (calculated if percent present).
   */
  @visibility(Lifecycle.Read)
  @summary("Amount in the currency of the invoice")
  amount: Money;

  /**
   * The type of the discount.
   */
  @visibility(Lifecycle.Read)
  type: InvoiceLineDiscountType.amount;
}
