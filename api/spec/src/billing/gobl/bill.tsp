import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace GOBL.Bill;

@summary("Invoice represents a document that is sent to a customer to request payment for goods or services.")
model Invoice {
  id: ULID;
  type?: InvoiceType;

  @summary("Used as a prefix to group codes.")
  series?: CBC.Code;

  @summary("Sequential code used to identify this invoice in tax declarations.")
  // NOTE: This is required in the original schema, but we can't enforce it here as we will only know this when the
  // invoice is issued.
  code?: CBC.Code;

  @summary("The time the invoice was issued.")
  // NOTE: This replaces the issueDate in the gobl schema, as date only is a bad practice and should be avoided.
  issuedAt?: DateTime;

  // Omitted fields (not required yet):
  //  op_date: Date when the operation defined by the invoice became effective.
  //  value_date: When the taxes of this invoice become accountable, if none set, the issue date is used.
  //  exchange_rates: Exchange rates to be used when converting the invoices monetary values into other currencies.
  //     -> We are not supporting multi-currency invoices yet, so this is not supported.
  //  charges: 	Charges or surcharges applied to the complete invoice.
  //     -> We are not supporting this yet.
  //  outlays: 	Expenses paid for by the supplier but invoiced directly to the customer.
  //     -> We are not supporting this yet.
  //  ordering: Ordering details including document references and buyer or seller parties.
  //     -> We are not supporting this yet (e.g. order handled by seperate entity, physical goods delivery details, etc.)
  //  delivery: Specific details on delivery of the goods referenced in the invoice.
  //     -> We are not supporting physical goods delivery yet.
  //  tax: Special tax configuration for billing.
  //     -> The prices include is the meaningful part, but we should make that an array, let's see if we need this as we implement more features.

  @summary("Currency fot all invoice totals.")
  currency?: Currency.Code;

  @summary("Key information regarding previous invoices and potentially details as to why they were corrected.")
  preceding?: Org.DocumentRef;

  @summary("The taxable entity supplying the goods or services.")
  supplier: Org.Party;

  @summary("Legal entity receiving the goods or services.")
  customer: Org.Party;

  @summary("List of invoice lines representing each of the items sold to the customer.")
  lines?: Line[];

  @summary("Discounts or allowances applied to the complete invoice.")
  discounts?: Discount[];

  @summary("Information on when, how, and to whom the invoice should be paid.")
  payment?: Payment;

  @summary("Summary of all the invoice totals, including taxes (calculated).")
  totals?: Totals;

  @summary("Unstructured information that is relevant to the invoice, such as correction or additional legal details.")
  // TODO: let's add the typed notes
  notes?: CBC.Note[];

  @summary("Additional semi-structured data that doesn’t fit into the body of the invoice.")
  meta?: CBC.Meta;
}

enum InvoiceType {
  @summary("A regular commercial invoice document between a supplier and customer.")
  standard: "standard",

  @summary("For a clients validation before sending a final invoice.")
  proforma: "proforma",

  @summary("Corrected invoice that completely *replaces* the preceding document.")
  corrected: "corrected",

  @summary("Reflects a refund either partial or complete of the preceding document. A credit note effectively *extends* the previous document.")
  credit_note: "credit-note",

  @summary("An additional set of charges to be added to the preceding document.")
  debit_note: "debit-note",

  @summary("Any other type of invoice that does not fit into the standard categories and implies\nthat any scenarios defined in tax regimes or addons will not be applied.\n\nThis is useful for being able to create invoices with custom types in extensions, but is not recommend for general use.")
  other: "other",
}

@summary("Discount represents an allowance applied to the complete document independent from the individual lines.")
model Discount {
  @summary("Universally Unique Identifier.")
  @format("uuid")
  uuid?: string;

  @summary("Line number inside the list of discounts (calculated)")
  i?: int32;

  @summary("Reference or ID for this Discount")
  ref?: string;

  @summary("Base represents the value used as a base for percent calculations instead of the invoice's sum of lines.")
  base?: Numeric;

  @summary("Percentage to apply to the base or invoice's sum.")
  percent?: Percentage;

  @summary("Amount to apply (calculated if percent present).")
  amount: Numeric;

  @summary("List of taxes to apply to the discount")
  taxes?: Tax.Combo[];

  @summary("Code for the reason this discount applied")
  code?: string;

  @summary("Text description as to why the discount was applied")
  reason?: string;

  @summary("Additional semi-structured information.")
  meta?: CBC.Meta;
}

model InvoiceItem {
  @summary("Line number inside the parent (calculated)")
  @visibility("read")
  @encodedName("application/json", "i")
  index: int32;

  quantity: Numeric;

  @visibility("read")
  sum: Numeric;

  discounts: LineDiscount[];
  charges: LineCharge[];
  taxes: GOBL.Tax.Combo[];
  total: Numeric;
  notes: GOBL.CBC.Note[];
}

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
}

@extension("x-go-type", "bill.LineDiscount")
@extension(
  "x-go-type-import",
  {
    path: "github.com/invopop/gobl/bill",
  }
)
@summary("LineDiscount represents an amount deducted from the line, and will be applied before taxes.")
model LineDiscount {
  ...GenericCharge;
}

@extension("x-go-type", "bill.LineCharge")
@extension(
  "x-go-type-import",
  {
    path: "github.com/invopop/gobl/bill",
  }
)
@summary("LineCharge represents an amount added to the line, and will be applied before taxes.")
model LineCharge {
  ...GenericCharge;
}

/**
 * Omitted fields:
 * - taxes: Summary of all the taxes included in the invoice.
 *    We don't need that detailed information here, as we only need the total tax amount.
 */
@summary("Totals contains the summaries of all calculations for the invoice.")
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

  @summary("Total paid in outlays that need to be reimbursed")
  outlays?: Numeric;

  @summary("Total amount to be paid after applying taxes and outlays.")
  payable: Numeric;

  @summary("Total amount already paid in advance.")
  advance?: Numeric;

  @summary("How much actually needs to be paid now.")
  due?: Numeric;
}

/**
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
model Payment {
  terms?: pay.Terms;
}
