import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace Invoices;

@pattern("^(?:[a-z]|[a-z0-9][a-z0-9-+]*[a-z0-9])$")
@minLength(1)
@maxLength(64)
@friendlyName("BillingCBCKey")
scalar Key extends string;

@pattern("^[A-Za-z0-9]+([\\.\\-\\/ _]?[A-Za-z0-9]+)*$")
@minLength(1)
@maxLength(32)
@friendlyName("BillingCBCCode")
scalar Code extends string;

// TODO: This is an enum, coming from code.json, we should use the json schema to have the available values
@minLength(3)
@maxLength(3)
@friendlyName("BillingCurrencyCode")
scalar CurrencyCode extends string;

// TODO: This is an enum, coming from tax-country-code.json, we should use the json schema to have the available values
@minLength(2)
@maxLength(2)
@friendlyName("BillingTaxCountryCode")
scalar TaxCountryCode extends string;

// TODO: This is an enum, coming from iso-country-code.json, we should use the json schema to have the available values
@minLength(2)
@maxLength(2)
@friendlyName("BillingISOCountryCode")
scalar ISOCountryCode extends string;

// TODO: we need to have a unit database
@summary("Unit describes how the quantity of the product should be interpreted either using a GOBL key, or UN/ECE code.")
scalar Unit extends string;
