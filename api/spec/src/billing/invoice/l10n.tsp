import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace Invoice.L10n;

// TODO: This is an enum, coming from tax-country-code.json, we should use the json schema to have the available values
@minLength(2)
@maxLength(2)
scalar TaxCountryCode extends string;

// TODO: This is an enum, coming from iso-country-code.json, we should use the json schema to have the available values
@minLength(2)
@maxLength(2)
scalar ISOCountryCode extends string;
