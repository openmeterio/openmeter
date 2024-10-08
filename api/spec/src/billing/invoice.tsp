import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter.Billing;

@summary("An invoice")
// @extension("x-go-type", "Invoice")
model Invoice {
  charge: InvoiceCharge;
}

model InvoiceCharge {
  key: CBCKey;
  i?: int32;

  @summary("Reference, code used to refer to this charge")
  ref?: string;
}
