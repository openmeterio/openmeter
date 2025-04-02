import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

namespace Invoices;

/**
 * InvoiceGenericCharge represents a charge or discount that can be applied to a line or the entire invoice.
 */
@friendlyName("InvoiceGenericCharge")
model InvoiceGenericCharge {
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
  code?: string;

  /**
   * Text description as to why the discount was applied.
   */
  @visibility(Lifecycle.Read)
  description?: string;
}

/**
 * InvoiceLineGenericAmountCharge represents a charge or discount that can be applied to a line.
 *
 * The charge is an amount that is added or deducted from the line, and will be applied before taxes.
 */
@friendlyName("InvoiceLineGenericAmountCharge")
model InvoiceLineGenericAmountCharge {
  ...InvoiceGenericCharge;

  /**
   * Fixed discount amount to apply (calculated if percent present).
   */
  @visibility(Lifecycle.Read)
  @summary("Amount in the currency of the invoice")
  amount: Money;
}

/**
 * InvoiceLineGenericUsageCharge represents a charge or discount that can be applied to a line.
 *
 * The charge is a usage that is added or deducted from the line, and will be applied before taxes.
 */
@friendlyName("InvoiceLineGenericUsageCharge")
model InvoiceLineGenericUsageCharge {
  ...InvoiceGenericCharge;

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
  ...InvoiceLineGenericUsageCharge;

  /**
   * The type of the discount.
   */
  @visibility(Lifecycle.Read)
  type: InvoiceLineDiscountType.usage;

  /**
   * The discount from the rate card this discount is based on.
   */
  @visibility(Lifecycle.Read)
  rateCardDiscount?: OpenMeter.ProductCatalog.Discount;
}

/**
 * InvoiceLineDiscountAmount represents an amount deducted from the line, and will be applied before taxes.
 */
@friendlyName("InvoiceLineDiscountAmount")
model InvoiceLineDiscountAmount {
  ...InvoiceLineGenericAmountCharge;

  /**
   * The type of the discount.
   */
  @visibility(Lifecycle.Read)
  type: InvoiceLineDiscountType.amount;

  /**
   * The discount from the rate card this discount is based on.
   */
  @visibility(Lifecycle.Read)
  rateCardDiscount?: OpenMeter.ProductCatalog.Discount;

  /**
   * External IDs of the invoice in other apps such as Stripe.
   */
  @visibility(Lifecycle.Read)
  externalIds?: InvoiceLineAppExternalIds;
}

/**
 * The type of the charge on the invoice line.
 */
@friendlyName("InvoiceLineChargeType")
enum InvoiceLineChargeType {
  amount: "amount",
  //  usage: "usage",
}

/**
 * InvoiceLineCharge represents a charge on the invoice line.
 */
@friendlyName("InvoiceLineCharge")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union InvoiceLineCharge {
  amount: InvoiceLineChargeAmount,
  //  usage: InvoiceLineChargeUsage,
}

/**
 * InvoiceLineChargeAmount represents an amount added to the line, and will be applied before taxes.
 */
@friendlyName("InvoiceLineChargeAmount")
model InvoiceLineChargeAmount {
  ...InvoiceLineGenericAmountCharge;

  /**
   * The type of the charge.
   */
  @visibility(Lifecycle.Read)
  type: InvoiceLineChargeType.amount;
}
