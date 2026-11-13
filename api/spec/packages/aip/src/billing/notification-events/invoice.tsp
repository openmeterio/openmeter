import "@typespec/json-schema";
import "@openmeter/typespec-notification-events";
import "../../shared/index.tsp";
import "../../invoices/invoice.tsp";

using TypeSpec.JsonSchema;
using NotificationCodegen;

/**
 * Notification Event contracts owned by the billing domain.
 */
namespace Billing.NotificationEvents;

/**
 * Payload of `v1.billing.invoice.created`.
 *
 * The invoice has the same shape as the v3 invoice API returns. A change to
 * that API changes this contract, so definitions.sum fails until the change is
 * either accepted as a new Event type version or this payload is frozen into
 * dedicated models.
 */
@jsonSchema
@id("v1.billing.invoice.created.payload")
@eventType("v1.billing.invoice.created")
@friendlyName("BillingInvoiceCreatedV1Payload")
model InvoiceCreatedV1Payload {
  /**
   * The invoice as it was created.
   */
  invoice: Invoices.InvoiceStandard;
}

/**
 * Payload of `v1.billing.invoice.updated`.
 *
 * Shares the v3 invoice shape with `v1.billing.invoice.created`.
 */
@jsonSchema
@id("v1.billing.invoice.updated.payload")
@eventType("v1.billing.invoice.updated")
@friendlyName("BillingInvoiceUpdatedV1Payload")
model InvoiceUpdatedV1Payload {
  /**
   * The invoice after the update.
   */
  invoice: Invoices.InvoiceStandard;
}
