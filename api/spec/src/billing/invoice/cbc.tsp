import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace Invoice.CBC;

@pattern("^(?:[a-z]|[a-z0-9][a-z0-9-+]*[a-z0-9])$")
@minLength(1)
@maxLength(64)
@friendlyName("BillingCBCKey")
scalar Key extends string;

@pattern("^(?:[a-z]|[a-z0-9][a-z0-9-+]*[a-z0-9])$")
@friendlyName("BillingCBCMetaKey")
scalar MetaKey extends string;

@friendlyName("BillingCBCMeta")
model Meta {
  ...Record<MetaKey>;
}

@pattern("^[A-Za-z0-9]+([\\.\\-\\/ _]?[A-Za-z0-9]+)*$")
@minLength(1)
@maxLength(32)
@friendlyName("BillingCBCCode")
scalar Code extends string;

@friendlyName("BillingNoteKey")
enum NoteKey {
  goods: "goods",
  payment: "payment",
  legal: "legal",
  dangerous_goods: "dangerous-goods",
  ack: "ack",
  rate: "rate",
  reason: "reason",
  dispute: "dispute",
  customer: "customer",
  glossary: "glossary",
  customs: "customs",
  general: "general",
  handling: "handling",
  packaging: "packaging",
  loading: "loading",
  price: "price",
  priority: "priority",
  regulatory: "regulatory",
  safety: "safety",
  ship_line: "ship-line",
  supplier: "supplier",
  transport: "transport",
  delivery: "delivery",
  quarantine: "quarantine",
  tax: "tax",
}

@friendlyName("BillingInvoiceNote")
model Note {
  @summary("Key specifying subject of the text")
  key?: NoteKey;

  @summary("Code used for additional data that may be required to identify the note.")
  code?: string;

  @summary("Source of this note, especially useful when auto-generated.")
  src?: Key;

  @summary("The contents of the note")
  text: string;

  @summary("Additional information about the note")
  meta?: CBC.Meta;
}
