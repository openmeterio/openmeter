import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace GOBL.L10n;

// TODO: This is an enum, coming from tax-country-code.json
@minLength(2)
@maxLength(2)
scalar TaxCountryCode extends string;
