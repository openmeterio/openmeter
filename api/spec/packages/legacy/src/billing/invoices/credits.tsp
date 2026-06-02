import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

namespace OpenMeter;

/**
 * InvoiceLineCreditAllocation represents a credit amount allocated to the line before taxes are applied.
 */
@friendlyName("InvoiceLineCreditAllocation")
model InvoiceLineCreditAllocation {
  /**
   * Amount allocated from credits.
   */
  @visibility(Lifecycle.Read)
  @summary("Amount in the currency of the invoice")
  amount: Money;

  /**
   * Text description as to why the credit was allocated.
   */
  @visibility(Lifecycle.Read)
  description?: string;
}
