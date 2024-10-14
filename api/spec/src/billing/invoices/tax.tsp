import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace Invoices;

@friendlyName("BillingTaxItem")
model TaxItem {
  /*
   * TODO: This should match the description from the plan (per backend tax items).
   * @encodedName("application/json", "cat")
   * @summary("Tax category code from those available inside a region.")
   * category: Invoice.CBC.Key;
   */

  @summary("Percent defines the percentage set manually or determined from the rate\nkey (calculated if rate present). A nil percent implies that this tax combo\nis **exempt** from tax.")
  percent?: Percentage;

  @summary("Some countries require an additional surcharge (calculated if rate present).")
  surcharge?: Numeric;

  @summary("Is the tax item inclusive or exclusive of the base amount.")
  behavior?: TaxBehavior;
}

@friendlyName("BillingTaxBehavior")
enum TaxBehavior {
  @summary("Tax is included in the base amount.")
  inclusive: "inclusive",

  @summary("Tax is added to the base amount.")
  exclusive: "exclusive",
}

@friendlyName("BillingTaxIdentity")
@summary("Identity stores the details required to identify an entity for tax purposes in a specific country.")
model TaxIdentity {
  @summary("Tax country code for Where the tax identity was issued.")
  country: TaxCountryCode;

  @summary("Normalized code shown on the original identity document.")
  code?: Code;
}
