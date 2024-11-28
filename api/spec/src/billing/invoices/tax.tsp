import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;
using OpenMeter.ProductCatalog;

namespace Invoices;

/**
 * TaxConfig stores the configuration for a tax line relative to an invoice line.
 */
@friendlyName("InvoiceLineTaxItem")
model InvoiceLineTaxItem {
  /**
   * Tax provider configuration.
   */
  @visibility("read")
  config?: TaxConfig;

  /**
   * Percent defines the percentage set manually or determined from
   * the rate key (calculated if rate present). A nil percent implies that
   * this tax combo is **exempt** from tax.")
   */
  @visibility("read")
  percent?: Percentage;

  /**
   * Some countries require an additional surcharge (calculated if rate present).
   */
  @visibility("read")
  surcharge?: Numeric;

  /**
   * Is the tax item inclusive or exclusive of the base amount.
   */
  @visibility("read")
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
