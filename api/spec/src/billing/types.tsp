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

enum InvoiceStatus {
  pending_creation: "pending_creation",
  created: "created",
  draft: "draft",
  draft_sync: "draft_sync",
  draft_sync_failed: "draft_sync_failed",
  issuing: "issuing",
  issued: "issued",
  issuing_failed: "issuing_failed",
  manual_approval_needed: "manual_approval_needed",
  deleted: "deleted",
  validation_failed: "validation_failed",
}
