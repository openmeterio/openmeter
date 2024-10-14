import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace Invoices;

@summary("Invoice summary represents an extract of the invoice object")
@friendlyName("BillingInvoiceSummary")
model InvoiceSummary {
  @visibility("read", "query", "update")
  id: ULID;

  @visibility("read", "query")
  type?: InvoiceType;

  @summary("The taxable entity supplying the goods or services.")
  @visibility("read", "query", "update")
  supplier: Party;

  @summary("Legal entity receiving the goods or services.")
  @visibility("read", "query", "update")
  customer: Party;

  @summary("Used as a prefix to group codes.")
  @visibility("read", "query")
  series?: Code;

  @summary("Sequential code used to identify this invoice in tax declarations.")
  // NOTE: This is required in the original schema, but we can't enforce it here as we will only know this when the
  // invoice is issued.
  @visibility("read", "query")
  code?: Code;

  @summary("Currency for all invoice totals.")
  @visibility("read", "query")
  currency?: CurrencyCode;

  @summary("Key information regarding previous invoices and potentially details as to why they were corrected.")
  @visibility("read", "query") // We only populate this for voided invoices
  preceding?: DocumentRef;

  @summary("Summary of all the invoice totals, including taxes (calculated).")
  @visibility("read", "query")
  totals?: Totals;

  @visibility("read", "query")
  @summary("The status of the invoice.")
  status: InvoiceStatus;

  /**
   * The time the invoice was issued.
   *
   * Depending on the status of the invoice this can mean multiple things:
   * - draft, gathering: The time the invoice will be issued based on the workflow settings.
   * - issued: The time the invoice was issued.
   */
  @summary("The time the invoice was issued.")
  @visibility("read", "query")
  issuedAt?: DateTime;

  @summary("The workflow settings associated with this invoice")
  @visibility("read", "query", "update")
  workflow?: InvoiceWorkflowSettings;
}

/* Omitted fields (not required yet):
 * op_date: Date when the operation defined by the invoice became effective.
 * value_date: When the taxes of this invoice become accountable, if none set, the issue date is used.
 * exchange_rates: Exchange rates to be used when converting the invoices monetary values into other currencies.
 *    -> We are not supporting multi-currency invoices yet, so this is not supported.
 * charges: 	Charges or surcharges applied to the complete invoice.
 *    -> We are not supporting this yet.
 * outlays: 	Expenses paid for by the supplier but invoiced directly to the customer.
 *    -> We are not supporting this yet.
 * ordering: Ordering details including document references and buyer or seller parties.
 *    -> We are not supporting this yet (e.g. order handled by seperate entity, physical goods delivery details, etc.)
 * delivery: Specific details on delivery of the goods referenced in the invoice.
 *    -> We are not supporting physical goods delivery yet.
 * tax: Special tax configuration for billing.
 *    -> The prices include is the meaningful part, but we should make that an array, let's see if we need this as we implement more features.
 */
@summary("Invoice represents a document that is sent to a customer to request payment for goods or services.")
@friendlyName("BillingInvoice")
model Invoice {
  ...InvoiceSummary;

  @summary("List of invoice lines representing each of the items sold to the customer.")
  @visibility("read", "query") // Seperate API endpoint is available for updating
  lines?: Line[];

  @summary("Discounts or allowances applied to the complete invoice.")
  @visibility("read", "query") // Seperate API endpoint is available for updating
  discounts?: Discount[];

  @summary("Information on when, how, and to whom the invoice should be paid.")
  @visibility("read", "query") // Seperate API endpoint is available for updating
  payment?: Payment;
}

@friendlyName("BillingInvoiceWorkflowSettings")
model InvoiceWorkflowSettings {
  /**
   * updatedAt is the time when the invoice workflow was last updated.
   */
  @summary("Should the invoice auto advance to the next status?")
  autoAdvance: boolean;

  /**
   * The invoice stays in the draft status until this time.
   *
   * If auto advance is disabled the value will be empty.
   */
  draftUtil?: DateTime;

  /**
   * The period after which the invoice is due.
   *
   * default: 7 days
   */
  @encode(DurationKnownEncoding.ISO8601)
  @example(duration.fromISO("P1D"))
  dueAfter?: duration;
}

/**
 * InvoiceStatus describes the status of an invoice.
 */
@friendlyName("BillingInvoiceStatus")
enum InvoiceStatus {
  /**
   * The list of line items for the next invoice is being gathered.
   */
  gathering: "gathering",

  /**
   * The invoice is waiting for review by our partner.
   */
  review: "review",

  /**
   * The invoice has been issued to the customer.
   */
  issued: "issued",

  /**
   * The invoice has been paid by the customer.
   */
  paymentReceived: "payment_received",

  /**
   * A manual approval is required before the invoice can be issued.
   */
  manualApprovalRequired: "manual_approval_required",

  /**
   * There's a validation issue with the invoice.
   */
  validationFailed: "validation_failed",
}

/*
 * Omitted types: proforma, corrected, debit-note, other
 */
@friendlyName("BillingInvoiceType")
enum InvoiceType {
  @summary("A regular commercial invoice document between a supplier and customer.")
  standard: "standard",

  @summary("Reflects a refund either partial or complete of the preceding document. A credit note effectively *extends* the previous document.")
  credit_note: "credit-note",
}

@friendlyName("BillingInvoiceDiscount")
@summary("Discount represents an allowance applied to the complete document independent from the individual lines.")
model Discount {
  @summary("Base represents the value used as a base for percent calculations instead of the invoice's sum of lines.")
  base?: Numeric;

  @summary("Percentage to apply to the base or invoice's sum.")
  percent?: Percentage;

  @summary("Amount to apply (calculated if percent present).")
  amount: Numeric;

  @summary("Code for the reason this discount applied")
  code?: string;

  @summary("Text description as to why the discount was applied")
  reason?: string;

  // TODO: Add reference fields to plan/subscriptions if the discount is coming from there.
}

alias InvoiceLine = UnitPriceLine;

@summary("LineTypes represents the different types of lines that can be used in an invoice.")
@friendlyName("BillingLineTypes")
enum LineTypes {
  /*
   * The line represents a product or service that is sold to the customer.
   *
   * If the item is a single charge or based on a metered value it doesn't matter from the invoicing perspective.
   */
  unitPrice: "unit_price",
}

/**
 * Line represents a single item or service sold to the customer.
 *
 * This is a base class for all line types, and should not be used directly.
 * */
@discriminator("type")
@friendlyName("BillingLine")
model Line {
  @visibility("read", "query", "update")
  id: ULID;

  type: LineTypes;

  @summary("Discounts applied to this line.")
  discounts?: LineDiscount[];

  @summary("Charges applied to this line.")
  charges?: LineCharge[];

  @summary("Map of taxes to be applied and used in the invoice totals.")
  @visibility("read", "query") // Tax is handled by the tax provider, we only need to show the output of that here
  taxes?: TaxItem[];

  @visibility("read")
  @summary("Total sum of the line, including discounts and charges.")
  total: Numeric;

  @summary("Period of the line item applies to for revenue recognition pruposes.")
  period: Period;
}

@summary("Period represents a time range.")
@friendlyName("BillingPeriod")
model Period {
  @summary("Start of the period.")
  start: DateTime;

  @summary("End of the period.")
  end: DateTime;
}

/**
 * UnitPriceLine represents a line item that is sold to the customer based on a unit price.
 *
 * This is analogous to gobl's Line model, with the addition of the type and some metadata fields.
 */
@friendlyName("BillingUnitPriceLine")
model UnitPriceLine extends Line {
  type: LineTypes.unitPrice;

  @summary("Details about what is being sold.")
  item: UnitPriceItem;

  @summary("Quantity of the item being sold.")
  quantity: Numeric;
}

@summary("GenericCharge represents a charge or discount that can be applied to a line or the entire invoice.")
@friendlyName("BillingGenericCharge")
model GenericCharge {
  /**
   * Percentage if fixed amount not applied
   */
  @summary("Percent")
  percent?: Percentage;

  /**
   * Fixed discount amount to apply (calculated if percent present).
   */
  @summary("Amount")
  amount: Numeric;

  @summary("Reason code.")
  code?: string;

  @summary("Text description as to why the discount was applied.")
  reason?: string;

  // TODO: Add fields required for referencing the charge/discount in the plan if it is coming from
  // there.
}

@summary("LineDiscount represents an amount deducted from the line, and will be applied before taxes.")
@friendlyName("BillingLineDiscount")
model LineDiscount {
  ...GenericCharge;
}

@summary("LineCharge represents an amount added to the line, and will be applied before taxes.")
@friendlyName("BillingLineCharge")
model LineCharge {
  ...GenericCharge;
}

/*
 * Omitted fields:
 * - taxes: Summary of all the taxes included in the invoice.
 *    We don't need that detailed information here, as we only need the total tax amount.
 */
@summary("Totals contains the summaries of all calculations for the invoice.")
@friendlyName("BillingInvoiceTotals")
model Totals {
  @summary("Sum of all line item sums")
  sum: Numeric;

  @summary("Sum of all document level discounts")
  discount?: Numeric;

  @summary("Sum of all document level charges")
  charge?: Numeric;

  @summary("If prices include tax, this is the total tax included in the price.")
  taxIncluded?: Numeric;

  @summary("Sum of all line sums minus the discounts, plus the charges, without tax.")
  total: Numeric;

  @summary("Total amount of tax to apply to the invoice.")
  tax?: Numeric;

  @summary("Grand total after all taxes have been applied.")
  totalWithTax: Numeric;

  @summary("Rounding amount to apply to the invoice in case the total and payable amounts don't quite match.")
  rounding?: Numeric;

  @summary("Total amount to be paid after applying taxes and outlays.")
  payable: Numeric;

  @summary("Total amount already paid in advance.")
  advance?: Numeric;

  @summary("How much actually needs to be paid now.")
  due?: Numeric;
}

/*
 * Omitted fields:
 * - payee: The party responsible for receiving payment of the invoice, if not the supplier.
 *    We don't need this information here, as we only need the payment terms.
 * - advances: Any amounts that have been paid in advance and should be deducted from the amount due.
 *    We are not supporting prepayments yet.
 * - instructions: Details on how payment should be made.
 *    This is deferred to the payment provider for now, if we start supporing more payment providers,
 *    we might need this information.
 */
@summary("Payment contains details as to how the invoice should be paid.")
@friendlyName("BillingInvoicePayment")
model Payment {
  terms?: PaymentTerms;
}
