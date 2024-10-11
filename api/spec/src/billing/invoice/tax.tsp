import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace Invoice.Tax;

@friendlyName("BillingTaxCombo")
model Combo {
  @encodedName("application/json", "cat")
  @summary("Tax category code from those available inside a region.")
  category: Invoice.CBC.Key;

  @summary("Country code override when issuing with taxes applied from different countries.")
  country: Invoice.L10n.TaxCountryCode;

  @summary("Rate within a category to apply.")
  rate: Invoice.CBC.Key;

  @summary("Percent defines the percentage set manually or determined from the rate\nkey (calculated if rate present). A nil percent implies that this tax combo\nis **exempt** from tax.")
  percent?: Percentage;

  @summary("Some countries require an additional surcharge (calculated if rate present).")
  surcharge?: Numeric;

  @summary("Local codes that apply for a given rate or percentage that need to be identified and validated.")
  @encodedName("application/json", "ext")
  extensions: Invoice.CBC.Meta;
}

@friendlyName("BillingTaxIdentity")
@summary("Identity stores the details required to identify an entity for tax purposes in a specific country.")
model Identity {
  @summary("Tax country code for Where the tax identity was issued.")
  country: L10n.TaxCountryCode;

  @summary("Normalized code shown on the original identity document.")
  code?: CBC.Code;

  @summary("Type is set according to the requirements of each regime, some have a single tax document type code, others require a choice to be made. Deprecated: Tax Identities should only be used for VAT or similar codes for companies. Use the identities array for other types of identification.")
  type?: CBC.Key;
}
