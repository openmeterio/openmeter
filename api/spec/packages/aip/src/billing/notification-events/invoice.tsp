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
 * Period an invoice covers.
 */
@friendlyName("BillingInvoiceSnapshotPeriod")
model InvoiceSnapshotPeriod {
  /**
   * Start of the period, inclusive.
   */
  from: Shared.DateTime;

  /**
   * End of the period, exclusive.
   */
  to: Shared.DateTime;
}

/**
 * Key facts of an invoice at the time of the Event.
 *
 * The snapshot carries what identifies the invoice and what most consumers act
 * on. Consumers fetch the full invoice through the API when they need lines or
 * other details. The fetched invoice can be newer than the Event; compare
 * `updated_at` to tell.
 */
@friendlyName("BillingInvoiceSnapshot")
model InvoiceSnapshot {
  /**
   * Invoice ID.
   */
  id: Shared.ULID;

  /**
   * Human-readable invoice number.
   */
  number: string;

  /**
   * Invoice type. Known values: `standard`.
   *
   * New types can be added without a new Event type version, so consumers
   * must handle values they do not know.
   */
  type: string;

  /**
   * Invoice status. Known values: `draft`, `issuing`, `issued`,
   * `payment_processing`, `overdue`, `paid`, `uncollectible`, `voided`.
   *
   * New statuses can be added without a new Event type version, so consumers
   * must handle values they do not know.
   */
  status: string;

  /**
   * Customer the invoice belongs to.
   */
  customer_id: Shared.ULID;

  /**
   * Key of the customer, if the customer has one.
   */
  customer_key?: Shared.ExternalResourceKey;

  /**
   * Invoice currency.
   */
  currency: Shared.CurrencyCode;

  /**
   * Total amount of the invoice, including taxes, discounts and credits.
   */
  total: Shared.Numeric;

  /**
   * Service period covered by the invoice, once it is known.
   */
  service_period?: InvoiceSnapshotPeriod;

  /**
   * When the invoice was issued to the customer.
   */
  issued_at?: Shared.DateTime;

  /**
   * When payment is due.
   */
  due_at?: Shared.DateTime;

  /**
   * When the invoice was created.
   */
  created_at: Shared.DateTime;

  /**
   * When the invoice was last updated.
   */
  updated_at: Shared.DateTime;
}

/**
 * Payload of `v1.billing.invoice.created`.
 */
@jsonSchema
@id("v1.billing.invoice.created.payload")
@eventType("v1.billing.invoice.created")
@friendlyName("BillingInvoiceCreatedV1Payload")
model InvoiceCreatedV1Payload {
  /**
   * The invoice as it was created.
   */
  invoice: InvoiceSnapshot;
}

/**
 * Payload of `v1.billing.invoice.updated`.
 */
@jsonSchema
@id("v1.billing.invoice.updated.payload")
@eventType("v1.billing.invoice.updated")
@friendlyName("BillingInvoiceUpdatedV1Payload")
model InvoiceUpdatedV1Payload {
  /**
   * The invoice after the update.
   */
  invoice: InvoiceSnapshot;
}
