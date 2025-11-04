import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";

import "../errors.tsp";
import "../types.tsp";
import "../billing/invoices/main.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;

/**
 * Custom Invoicing app can be used for interface with any invoicing or payment system.
 *
 * This app provides ways to manipulate invoices and payments, however the integration
 * must rely on Notifications API to get notified about invoice changes.
 */
@friendlyName("CustomInvoicingApp")
model CustomInvoicingApp {
  ...AppBase;

  /**
   * The app's type is CustomInvoicing.
   */
  type: AppType.CustomInvoicing;

  /**
   * Enable draft.sync hook.
   *
   * If the hook is not enabled, the invoice will be progressed to the next state automatically.
   */
  enableDraftSyncHook: boolean;

  /**
   * Enable issuing.sync hook.
   *
   * If the hook is not enabled, the invoice will be progressed to the next state automatically.
   */
  enableIssuingSyncHook: boolean;
}

@tag("App: Custom Invoicing")
@friendlyName("AppCustomInvoicing")
interface AppCustomInvoicingEndpoints {
  @post
  @route("/api/v1/apps/custom-invoicing/{invoiceId}/draft/synchronized")
  @operationId("appCustomInvoicingDraftSynchronized")
  @summary("Submit draft synchronization results")
  draftSyncronized(
    @path invoiceId: ULID,
    @body body: CustomInvoicingDraftSynchronizedRequest,
  ): void | CommonErrors;

  @post
  @route("/api/v1/apps/custom-invoicing/{invoiceId}/issuing/synchronized")
  @operationId("appCustomInvoicingIssuingSynchronized")
  @summary("Submit issuing synchronization results")
  finalized(
    @path invoiceId: ULID,
    @body body: CustomInvoicingFinalizedRequest,
  ): void | CommonErrors;

  @post
  @route("/api/v1/apps/custom-invoicing/{invoiceId}/payment/status")
  @operationId("appCustomInvoicingUpdatePaymentStatus")
  @summary("Update payment status")
  paymentStatus(
    @path invoiceId: ULID,
    @body body: CustomInvoicingUpdatePaymentStatusRequest,
  ): void | CommonErrors;
}

/**
 * Information to synchronize the invoice.
 *
 * Can be used to store external app's IDs on the invoice or lines.
 */
@friendlyName("CustomInvoicingSyncResult")
model CustomInvoicingSyncResult {
  /**
   * If set the invoice's number will be set to this value.
   */
  invoiceNumber?: InvoiceNumber;

  /**
   * If set the invoice's invoicing external ID will be set to this value.
   */
  externalId?: string;

  /**
   * If set the invoice's line external IDs will be set to this value.
   *
   * This can be used to reference the external system's entities in the
   * invoice.
   */
  lineExternalIds?: CustomInvoicingLineExternalIdMapping[];

  /**
   * If set the invoice's line discount external IDs will be set to this value.
   *
   * This can be used to reference the external system's entities in the
   * invoice.
   */
  lineDiscountExternalIds?: CustomInvoicingLineDiscountExternalIdMapping[];
}

/**
 * Mapping between lines and external IDs.
 */
@friendlyName("CustomInvoicingLineExternalIdMapping")
model CustomInvoicingLineExternalIdMapping {
  /**
   * The line ID.
   */
  lineId: ULID;

  /**
   * The external ID (e.g. custom invoicing system's ID).
   */
  externalId: string;
}

/**
 * Mapping between line discounts and external IDs.
 */
@friendlyName("CustomInvoicingLineDiscountExternalIdMapping")
model CustomInvoicingLineDiscountExternalIdMapping {
  /**
   * The line discount ID.
   */
  lineDiscountId: ULID;

  /**
   * The external ID (e.g. custom invoicing system's ID).
   */
  externalId: string;
}

/**
 * Information to finalize the payment details of an invoice.
 */
@friendlyName("CustomInvoicingFinalizedPaymentRequest")
model CustomInvoicingFinalizedPaymentRequest {
  /**
   * If set the invoice's payment external ID will be set to this value.
   */
  externalId?: string;
}

/**
 * Information to finalize the draft details of an invoice.
 */
@friendlyName("CustomInvoicingDraftSynchronizedRequest")
model CustomInvoicingDraftSynchronizedRequest {
  /**
   * The result of the synchronization.
   */
  invoicing?: CustomInvoicingSyncResult;
}

/**
 * Information to finalize the invoicing details of an invoice.
 */
@friendlyName("CustomInvoicingFinalizedInvoicingRequest")
model CustomInvoicingFinalizedInvoicingRequest {
  /**
   * If set the invoice's number will be set to this value.
   */
  invoiceNumber?: InvoiceNumber;

  /**
   * If set the invoice's sent to customer at will be set to this value.
   */
  sentToCustomerAt?: DateTime;
}

/**
 * Information to finalize the invoice.
 *
 * If invoicing.invoiceNumber is not set, then a new invoice number will be generated (INV- prefix).
 */
@friendlyName("CustomInvoicingFinalizedRequest")
model CustomInvoicingFinalizedRequest {
  /**
   * The result of the synchronization.
   */
  invoicing?: CustomInvoicingFinalizedInvoicingRequest;

  /**
   * The result of the payment synchronization.
   */
  payment?: CustomInvoicingFinalizedPaymentRequest;
}

/**
 * Payment trigger to execute on a finalized invoice.
 */
@friendlyName("CustomInvoicingPaymentTrigger")
enum CustomInvoicingPaymentTrigger {
  Paid: "paid",
  PaymentFailed: "payment_failed",
  PaymentUncollectible: "payment_uncollectible",
  PaymentOverdue: "payment_overdue",
  ActionRequired: "action_required",
  Void: "void",
}

/**
 * Update payment status request.
 *
 * Can be used to manipulate invoice's payment status (when custominvoicing app is being used).
 */
@friendlyName("CustomInvoicingUpdatePaymentStatusRequest")
model CustomInvoicingUpdatePaymentStatusRequest {
  /**
   * The trigger to be executed on the invoice.
   */
  trigger: CustomInvoicingPaymentTrigger;
}
