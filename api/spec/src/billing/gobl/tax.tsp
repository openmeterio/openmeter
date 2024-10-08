import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace GOBL.Tax;

model Combo {
  @encodedName("application/json", "cat")
  @summary("Tax category code from those available inside a region.")
  category: GOBL.CBC.Key;

  @summary("Country code override when issuing with taxes applied from different countries.")
  country: GOBL.L10n.TaxCountryCode;

  @summary("Rate within a category to apply.")
  rate: GOBL.CBC.Key;

  @summary("Percent defines the percentage set manually or determined from the rate\nkey (calculated if rate present). A nil percent implies that this tax combo\nis **exempt** from tax.")
  percent?: Percentage;

  @summary("Some countries require an additional surcharge (calculated if rate present).")
  surcharge?: Numeric;

  @summary("Local codes that apply for a given rate or percentage that need to be identified and validated.")
  @encodedName("application/json", "ext")
  extensions: GOBL.CBC.Meta;
}
