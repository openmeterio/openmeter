import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace Invoices;

@friendlyName("BillingTaxItem")
model TaxItem {
  @summary("Tax provider configuration.")
  config?: OpenMeter.Billing.TaxConfig;

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

@pattern("^[A-Za-z0-9]+([\\.\\-\\/ _]?[A-Za-z0-9]+)*$")
@minLength(1)
@maxLength(32)
@friendlyName("BillingTaxIdentificationCode")
scalar TaxIdentificationCode extends string;

@friendlyName("BillingTaxIdentity")
@summary("Identity stores the details required to identify an entity for tax purposes in a specific country.")
model TaxIdentity {
  @summary("Tax country code for Where the tax identity was issued.")
  country: CountryCode;

  @summary("Normalized tax code shown on the original identity document.")
  code?: TaxIdentificationCode;
}
