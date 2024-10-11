import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace Invoice.Currency;

// TODO: This is an enum, coming from code.json, we should use the json schema to have the available values
@minLength(3)
@maxLength(3)
@friendlyName("BillingCurrencyCode")
scalar Code extends string;
