import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter.Billing;

@route("/api/v1/billing/invoices")
@tag("Billing (Experimental)")
interface InvoicesAllCustomers {
  /**
   * List invoices
   */
  @get
  @summary("List invoices")
  listInvoices(
    @query
    @summary("Filter by the customer ID or key")
    customers?: Array<ULIDOrKey>,

    ...InvoiceListParams,
    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryLimitOffset,
    ...OpenMeter.QueryOrdering<ProfileOrderBy>,
  ): OpenMeter.PaginatedResponse<GOBL.Bill.Invoice> | OpenMeter.CommonErrors;
}

@route("/api/v1/billing/invoices/:customerIdOrKey/invoices")
@tag("Billing (Experimental)")
interface CustomerInvoices {
  /**
   * Create a new invoice from the pending items.
   *
   * This should be only called if for some reason we need to invoice a customer outside of the normal billing cycle.
   *
   * When creating an invoice, the pending items will be marked as invoiced and the invoice will be created with the total amount of the pending items.
   *
   * New pending items will be created for the period between now() and the next billing cycle's begining date for any metered item.
   */
  @post
  @summary("Create an invoice")
  create(
    @path
    customerIdOrKey: ULIDOrKey,

    @body
    request: InvoiceCreateRequest,
  ): {
    @statusCode _: 201;
    @body body: GOBL.Bill.Invoice;
  } | OpenMeter.CommonErrors;

  /**
   * List invoices
   */
  @get
  @summary("List invoices")
  listInvoicesByCustomerID(
    ...InvoiceListParams,
    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryLimitOffset,
    ...OpenMeter.QueryOrdering<ProfileOrderBy>,
  ): OpenMeter.PaginatedResponse<GOBL.Bill.Invoice> | OpenMeter.CommonErrors;

  /**
   * Get an invoice by ID.
   */
  @get
  @route("/:invoiceId")
  @summary("Get an invoice")
  getInvoiceByID(
    @path
    customerIdOrKey: ULIDOrKey,

    @path
    invoiceId: ULID,
  ): GOBL.Bill.Invoice | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Delete an invoice
   *
   * Only invoices that are in the draft (or earlier) status can be deleted.
   */
  @delete
  @route("/:invoiceId")
  @summary("Delete an invoice")
  deleteInvoice(
    @path
    customerIdOrKey: ULIDOrKey,

    @path
    invoiceId: ULID,
  ): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Recalculate an invoice's tax amounts (using the app set in the customer's billing profile)
   *
   * Note: charges might apply, depending on the tax provider.
   *
   */
  @post
  @route("/:invoiceId/tax/recalculate")
  @summary("Recalculate an invoice's tax amounts")
  recalculateTax(
    @path
    customerIdOrKey: ULIDOrKey,

    @path
    invoiceId: ULID,
  ): {
    @statusCode _: 200;
    @body body: GOBL.Bill.Invoice;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Send the invoice to the customer.
   *
   * This call instantly sends the invoice to the customer using the configured billing profile app.
   *
   * This call is valid in two invoice statuses:
   * - `draft`: the invoice will be sent to the customer, the invluce state becomes issued
   * - `manual_approval_needed`: the invoice will be sent to the customer, the invoice state becomes issued
   */
  @post
  @route("/:invoiceId/approve")
  @summary("Send the invoice to the customer")
  approve(
    @path
    customerIdOrKey: ULIDOrKey,

    @path
    invoiceId: ULID,
  ): {
    @statusCode _: 200;
    @body body: GOBL.Bill.Invoice;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Retry advancing the invoice's state to the next status.
   *
   * The call doesn't "approve the invoice", it only advances the invoice to the next status if the transition would be automatic.
   *
   * This call can be used to retry advancing the invoice's state to the next status if the previous attempt failed.
   *
   * This call is valid in the following invoice statuses:
   * - `draft_sync_failed`
   * - `issuing_failed`
   * - `validation_failed`
   */
  @post
  @route("/:invoiceId/retry")
  @summary("Retry advancing the invoice's state to the next status")
  retryStateTransitions(
    @path
    customerIdOrKey: ULIDOrKey,

    @path
    invoiceId: ULID,
  ): {
    @statusCode _: 200;
    @body body: GOBL.Bill.Invoice;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Void an invoice
   *
   * Only invoices that have been alread issued can be voided.
   *
   * Voiding an invoice will mark it as voided, the user can specify how to handle the voided items.
   */
  @post
  @route("/:invoiceId/void")
  @summary("Void an invoice")
  voidInvoice(
    @path
    customerIdOrKey: ULIDOrKey,

    @path
    invoiceId: ULID,

    @body
    request: InvoiceVoidRequest,
  ): {
    @statusCode _: 200;
    @body body: GOBL.Bill.Invoice;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Update invoice workflow
   *
   * This call can be used to update the invoice workflow, only those settings are adjustable
   * that are not referring to past activities (such as collection).
   */
  @put
  @route("/:invoiceId/workflow")
  @summary("Update invoice workflow")
  updateWorkflow(
    @path
    customerIdOrKey: ULIDOrKey,

    @path
    invoiceId: ULID,

    @body
    request: ExistingInvoiceWorkflow,
  ): {
    @statusCode _: 200;
    @body body: GOBL.Bill.Invoice;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Get the invoice workflow
   */
  @get
  @route("/:invoiceId/workflow")
  @summary("Get the invoice workflow")
  getWorkflow(
    @path
    customerIdOrKey: ULIDOrKey,

    @path
    invoiceId: ULID,
  ): ExistingInvoiceWorkflow | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

/**
 * Request to void an invoice
 */
model InvoiceVoidRequest {
  /**
   * The action to take on the voided items.
   */
  @summary("The action to take on the voided items")
  action: InvoiceVoidAction;

  /**
   * The reason for voiding the invoice.
   */
  @summary("The reason for voiding the invoice")
  reason: string;

  /**
   * Per line item overrides for the action.
   *
   * If not specified, the `action` will be applied to all items.
   */
  itemOverrides: InvoiceVoidItemOverride[];
}

model ExistingInvoiceWorkflow {
  /**
   * updatedAt is the time when the invoice workflow was last updated.
   *
   * This field is updated when the invoice workflow is updated, during put requests we
   * are validating the update time to ensure there is no parallel updates happening.
   */
  updatedAt: DateTime;

  @summary("Should the invoice auto advance to the next status?")
  autoAdvance: boolean;

  /**
   * The invoice stays in the draft status until this time.
   *
   * If auto advance is disabled the value will be empty.
   */
  draftUtil?: DateTime;

  /**
   * The period after which the invoice is due.
   *
   * default: 7 days
   */
  @encode(DurationKnownEncoding.ISO8601)
  @example(duration.fromISO("P1D"))
  dueAfter?: duration;
}

/**
 * InvoiceVoidAction describes how to handle the voided items.
 */
model InvoiceVoidAction {
  @summary("How much of the total items to be voided? (e.g. 100% means all charges are voided)")
  percentage: Percentage;

  @summary("How to handle the voided items, default: pending")
  action: InvoiceVoidItemAction;
}

enum InvoiceVoidItemAction {
  @summary("The items will never be charged for again")
  discard: "discard",

  @summary("Queue the items into the pending state, they will be included in the next invoice. (We want to generate an invoice right now)")
  pending: "pending",

  @summary("Queue the items into the pending state, they will be included in the next invoice, but not in the current one")
  pending_next_cycle: "pending_next_cycle",
}

/**
 * InvoiceVoidItemOverride describes how to handle a specific item in the invoice when voiding.
 */
model InvoiceVoidItemOverride {
  @summary("The item ID to override")
  itemId: ULID;

  @summary("The action to take on the item")
  action: InvoiceVoidItemAction;
}

/**
 * List/get is expeced to be perfomed on the main invoice object
 */
@route("/api/v1/billing/invoices/:customerIdOrKey/invoices/:invoiceId/items")
@tag("Billing (Experimental)")
interface InvoiceItems {
  /**
   * Delete an invoice item
   */
  @delete
  @route("/:itemId")
  @summary("Delete an invoice item")
  @operationId("deleteInvoiceItem")
  delete(@path customerIdOrKey: ULIDOrKey, @path invoiceId: ULID, @path itemId: ULID): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Update an invoice item
   */
  @put
  @route("/:itemId")
  @summary("Update an invoice item")
  @operationId("updateInvoiceItem")
  update(
    @path customerIdOrKey: ULIDOrKey,
    @path invoiceId: ULID,
    @path itemId: ULID,
    @body request: GOBL.Bill.InvoiceItem,
  ): {
    @statusCode _: 200;
    @body body: GOBL.Bill.InvoiceItem;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

@route("/api/v1/billing/invoices/:customerIdOrKey/invoices/:invoiceId/supplier")
@tag("Billing (Experimental)")
interface InvoiceSupplier {
  /**
   * Update/override the supplier information for the invoice
   *
   * Can be called only on invoices in the draft status.
   */
  @put
  @summary("Update the supplier information for the invoice")
  @operationId("updateInvoiceSupplier")
  update(
    @path
    customerIdOrKey: ULIDOrKey,

    @path
    invoiceId: ULID,

    @body
    request: InvoiceSupplierUpdateRequest,
  ): {
    @statusCode _: 200;
    // TODO: When we have gobl objects we should return an org.Party object
    // @body body: GOBL.Bill.InvoiceSupplier;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Syncronize the supplier from the billing profile
   */
  @post
  @route("/sync")
  @summary("Syncronize the supplier from the billing profile")
  @operationId("syncInvoiceSupplier")
  sync(
    @path
    customerIdOrKey: ULIDOrKey,

    @path
    invoiceId: ULID,
  ): {
    @statusCode _: 200;
    // TODO: When we have gobl objects we should return an org.Party object
    // @body body: GOBL.Bill.InvoiceSupplier;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

model InvoiceSupplierUpdateRequest {
  @summary("The supplier information to update")
  supplier: Supplier;
}

@route("/api/v1/billing/invoices/:customerIdOrKey/invoices/:invoiceId/customer")
@tag("Billing (Experimental)")
interface InvoiceCustomer {
  /**
   * Update/override the customer information for the invoice
   *
   * Can be called only on invoices in the draft status.
   */
  @put
  @summary("Update the customer information for the invoice")
  update(
    @path
    customerIdOrKey: ULIDOrKey,

    @path
    invoiceId: ULID,

    @body
    request: InvoiceCustomerUpdateRequest,
  ): {
    @statusCode _: 200;
    // TODO: when we have gobl types, we should return an org.Party object
    // @body body: GOBL.Bill.InvoiceCustomer;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Syncronize the customer from the customer entity
   */
  @post
  @route("/sync")
  @summary("Syncronize the customer from the billing profile")
  sync(
    @path
    customerIdOrKey: ULIDOrKey,

    @path
    invoiceId: ULID,
  ): {
    @statusCode _: 200;
    // TODO: When we have gobl objects we should return an org.Party object
    // @body body: GOBL.Bill.InvoiceCustomer;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

model InvoiceCustomerUpdateRequest {
  /**
   * The billing address of the customer.
   * Used for tax and invoicing.
   */
  @summary("The customer information to update")
  billingAddress?: Address;

  /**
   * The customer's name.
   */
  name?: string;

  /**
   * The customer's email.
   */
  primaryEmail?: string;
}

/**
 * TODO: discounts
 *  When we have that in the db let's add a similar api to the items' discunts
 */
@route("/api/v1/billing/invoices/:customerIdOrKey/invoices/:invoiceId/discounts")
@tag("Billing (Experimental)")
interface InvoiceDiscounts {}

/**
 * Common parameters for listing invoices
 */
model InvoiceListParams {
  @query
  @summary("Include the line item details into the response, or just the invoice header and totals")
  expandItems?: boolean;

  @query
  @summary("Filter by the invoice status")
  statuses?: Array<InvoiceStatus>;

  @query
  @summary("Filter by invoice creation time")
  issuedAfter?: DateTime;

  @query
  @summary("Filter by invoice creation time")
  issuedBefore?: DateTime;
}

model InvoiceCreateRequest {
  /**
   * The pending items to include in the invoice, if not provided all pending items will be included.
   */
  IncludePendingItems?: ULID[];
}
