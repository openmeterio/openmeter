import "@typespec/openapi3";

using TypeSpec.OpenAPI;

namespace OpenMeter.Billing;

/**
 * BillingAddress is an address with the required field for billing proposes
 */
@friendlyName("BillingAddress")
model BillingAddress {
  ...OmitProperties<Address, "country">;
  country: CountryCode;
}

/**
 * InvoiceID is the ID of an invoice.
 *
 * This model is used instead of a simple ULID to allow for extending the brief response with additional fields.
 */
model InvoiceSummary {
  /**
   * The invoice ID.
   */
  id: ULID;
}
