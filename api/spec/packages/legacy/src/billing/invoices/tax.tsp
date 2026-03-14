import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter;

/**
 * TaxConfig stores the configuration for a tax line relative to an invoice line.
 */
@friendlyName("InvoiceLineTaxItem")
model InvoiceLineTaxItem {
  /**
   * Tax provider configuration.
   */
  @visibility(Lifecycle.Read)
  config?: TaxConfig;

  /**
   * Percent defines the percentage set manually or determined from
   * the rate key (calculated if rate present). A nil percent implies that
   * this tax combo is **exempt** from tax.")
   */
  @visibility(Lifecycle.Read)
  percent?: Percentage;

  /**
   * Some countries require an additional surcharge (calculated if rate present).
   */
  @visibility(Lifecycle.Read)
  surcharge?: Numeric;

  /**
   * Is the tax item inclusive or exclusive of the base amount.
   */
  @visibility(Lifecycle.Read)
  behavior?: InvoiceLineTaxBehavior;
}

/**
 * InvoiceLineTaxBehavior details how the tax item is applied to the base amount.
 *
 * Inclusive means the tax is included in the base amount.
 * Exclusive means the tax is added to the base amount.
 */
@friendlyName("InvoiceLineTaxBehavior")
enum InvoiceLineTaxBehavior {
  /**
   * Tax is included in the base amount.
   */
  inclusive: "inclusive",

  /**
   * Tax is added to the base amount.
   */
  exclusive: "exclusive",
}
