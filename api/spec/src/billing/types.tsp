import "@typespec/openapi3";

using TypeSpec.OpenAPI;

/**
 * BillingAddress is an address with the required field for billing proposes
 */
@friendlyName("BillingAddress")
model BillingAddress {
  ...OmitProperties<Address, "country">;
  country: CountryCode;
}
