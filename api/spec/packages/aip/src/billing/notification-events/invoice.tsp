import "@typespec/json-schema";
import "@openmeter/typespec-notification-events";
import "../../shared/index.tsp";

using TypeSpec.JsonSchema;
using NotificationCodegen;

/**
 * Notification Event contracts owned by the billing domain.
 */
namespace Billing.NotificationEvents;

/**
 * Payload of `v1.billing.invoice.created`.
 *
 * The payload identifies the invoice and carries its state at creation time;
 * consumers fetch the full invoice through the API when they need it.
 */
@jsonSchema
@id("v1.billing.invoice.created.payload")
@eventType("v1.billing.invoice.created")
@friendlyName("BillingInvoiceCreatedV1Payload")
model InvoiceCreatedV1Payload {
  /**
   * Invoice that was created.
   */
  invoice_id: Shared.ULID;

  /**
   * Customer the invoice belongs to.
   */
  customer_id: Shared.ULID;

  /**
   * Invoice status at creation time.
   */
  status: string;

  /**
   * Invoice currency.
   */
  currency: Shared.CurrencyCode;

  /**
   * Invoice total at creation time.
   */
  total: Shared.Numeric;
}
