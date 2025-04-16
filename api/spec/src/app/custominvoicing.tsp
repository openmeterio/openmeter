import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";

import "../errors.tsp";
import "../types.tsp";
import "../billing/invoices/main.tsp";

using OpenMeter.App;
using OpenMeter.Billing.Invoice;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.App;

/**
 * Custom Invoicing app can be used for testing OpenMeter features.
 *
 * The app is intended to be used for custom invoicing integrations external to
 * openmeter.
 */
@friendlyName("CustomInvoicingApp")
model CustomInvoicingApp {
  ...AppBase;

  /**
   * The app's type is Custom Invoicing.
   */
  type: Type.CustomInvoicing;
}

@tag("Apps")
interface AppCustomInvoicing {
  @post
  @route("/api/v1/apps/custom-invoicing/{invoiceId}/draft/syncronized")
  @operationId("appCustomInvoicingDraftSyncronized")
  @summary("Draft syncronized")
  draftSyncronized(
    @path invoiceId: ULID,
    @body body: CustomInvoicingDraftSyncronizedRequest,
  ): void | CommonErrors;

  @post
  @route("/api/v1/apps/custom-invoicing/{invoiceId}/issuing/syncronized")
  @operationId("appCustomInvoicingIssuingSyncronized")
  @summary("Issuing syncronized")
  finalized(
    @path invoiceId: ULID,
    @body body: CustomInvoicingFinalizedRequest,
  ): void | CommonErrors;

  @post
  @route("/api/v1/apps/custom-invoicing/{invoiceId}/payment/status")
  @operationId("appCustomInvoicingPaymentStatus")
  @summary("Payment status")
  paymentStatus(
    @path invoiceId: ULID,
    @body body: CustomInvoicingUpdatePaymentStatusRequest,
  ): void | CommonErrors;
}

@friendlyName("CustomInvoicingValidationIssueInput")
model CustomInvoicingValidationIssueInput {
  /**
   * The severity of the issue.
   */
  @visibility(Lifecycle.Create)
  severity: ValidationIssueSeverity;

  /**
   * The field that the issue is related to, if available in JSON path format.
   */
  @visibility(Lifecycle.Create)
  field?: string;

  /**
   * Machine indentifiable code for the issue, if available.
   */
  @visibility(Lifecycle.Create)
  code?: string;

  /**
   * Additional context for the issue.
   */
  @visibility(Lifecycle.Create)
  metadata?: Metadata;
}

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

  /**
   * Validation issues found during the syncronization.
   */
  validationIssues?: CustomInvoicingValidationIssueInput[];
}

@friendlyName("CustomInvoicingLineExternalIdMapping")
model CustomInvoicingLineExternalIdMapping {
  lineId: ULID;
  externalId: string;
}

@friendlyName("CustomInvoicingLineDiscountExternalIdMapping")
model CustomInvoicingLineDiscountExternalIdMapping {
  lineDiscountId: ULID;
  externalId: string;
}

@friendlyName("CustomInvoicingPaymentSyncResult")
model CustomInvoicingPaymentSyncResult {
  /**
   * If set the invoice's payment external ID will be set to this value.
   */
  paymentExternalId?: string;

  /**
   * If set the invoice's sent to customer at will be set to this value.
   */
  sentToCustomerAt?: DateTime;

  /**
   * Validation issues found during the syncronization.
   */
  validationIssues?: CustomInvoicingValidationIssueInput[];
}

@friendlyName("CustomInvoicingDraftSyncronizedRequest")
model CustomInvoicingDraftSyncronizedRequest {
  /**
   * The result of the syncronization.
   */
  invoicing?: CustomInvoicingSyncResult;
}

@friendlyName("CustomInvoicingFinalizedRequest")
model CustomInvoicingFinalizedRequest {
  /**
   * The result of the syncronization.
   */
  invoicing?: CustomInvoicingSyncResult;

  /**
   * The result of the payment syncronization.
   */
  payment?: CustomInvoicingPaymentSyncResult;

  /**
   * The next status of the invoice.
   *
   * If not set, the custom integration is expected to call payment/status to
   * transition the invoice into the next state.
   */
  nextStatus?: CustomInvoicingUpdatePaymentStatusRequest;
}

@friendlyName("CustomInvoicingUpdatePaymentStatusRequest")
model CustomInvoicingUpdatePaymentStatusRequest {
  /**
   * The target status of the invoice.
   */
  targetStatus?: InvoiceStatus;

  /**
   * Validation issues found during the syncronization.
   */
  validationIssues?: CustomInvoicingValidationIssueInput[];
}
