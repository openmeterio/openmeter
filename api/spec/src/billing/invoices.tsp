using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.Billing;

/**
 * InvoicesEndpoints is a collection of endpoints that allow invoice operations without prior
 * knowledge of the invoice ID.
 */
@route("/api/v1/billing/invoices")
@tag("Billing (Experimental)")
interface InvoicesEndpoints {
  /**
   * Create a new invoice from the pending line items.
   *
   * This should be only called if for some reason we need to invoice a customer outside of the normal billing cycle.
   *
   * When creating an invoice, the pending line items will be marked as invoiced and the invoice will be created with the total amount of the pending items.
   *
   * New pending line items will be created for the period between now() and the next billing cycle's begining date for any metered item.
   *
   * The call can return multiple invoices if the pending line items are in different currencies.
   */
  @post
  @route("/invoice")
  @summary("Invoice a customer based on the pending line items")
  @operationId("invoicePendingLinesAction")
  @extension("x-internal", true)
  invoicePendingLinesAction(
    @body
    request: InvoicePendingLinesActionInput,
  ): {
    @statusCode _: 201;

    @body
    body: Invoices.Invoice[];
  } | OpenMeter.CommonErrors;

  /**
   * List invoices for a specific customer
   */
  @get
  @summary("List invoices")
  @operationId("listInvoices")
  @extension("x-internal", true)
  list(
    ...InvoiceListParams,
    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryLimitOffset,
    ...OpenMeter.QueryOrdering<InvoiceOrderBy>,
  ): OpenMeter.PaginatedResponse<Invoices.Invoice> | OpenMeter.CommonErrors;

  /**
   * Create a new pending line item (charge).
   *
   * This call is used to create a new pending line item for the customer if required a new
   * gathering invoice will be created.
   *
   * A new invoice will be created if:
   * - there is no invoice in gathering state
   * - the currency of the line item doesn't match the currency of any invoices in gathering state
   */
  @post
  @route("/lines")
  @summary("Create pending line items")
  @operationId("createPendingInvoiceLine")
  @extension("x-internal", true)
  createPendingInvoiceLine(
    @body
    request: Invoices.InvoicePendingLineCreate[],
  ): {
    @statusCode _: 201;
    @body body: Invoices.InvoiceLine[];
  } | OpenMeter.CommonErrors;
}

/**
 * CustomerInvoice is a collection of endpoints that allow operations on a specific invoice.
 */
@route("/api/v1/billing/invoices/{invoiceId}")
@tag("Billing (Experimental)")
interface InvoiceEndpoints {
  /**
   * Get an invoice by ID.
   */
  @get
  @summary("Get an invoice")
  @operationId("getInvoice")
  @extension("x-internal", true)
  getInvoice(
    @path
    invoiceId: ULID,

    @query
    expand: InvoiceExpand[] = #[InvoiceExpand.lines],

    @query
    includeDeletedLines: boolean = false,
  ): Invoices.Invoice | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Delete an invoice
   *
   * Only invoices that are in the draft (or earlier) status can be deleted.
   */
  @delete
  @summary("Delete an invoice")
  @operationId("deleteInvoice")
  @extension("x-internal", true)
  deleteInvoice(
    @path
    invoiceId: ULID,
  ): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Recalculate an invoice's tax amounts (using the app set in the customer's billing profile)
   *
   * Note: charges might apply, depending on the tax provider.
   */
  @post
  @route("/tax/recalculate")
  @summary("Recalculate an invoice's tax amounts")
  @operationId("recalculateInvoiceTaxAction")
  @extension("x-internal", true)
  recalculateTaxAction(
    @path
    invoiceId: ULID,
  ): Invoices.Invoice | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Approve an invoice and start executing the payment workflow.
   *
   * This call instantly sends the invoice to the customer using the configured billing profile app.
   *
   * This call is valid in two invoice statuses:
   * - `draft`: the invoice will be sent to the customer, the invluce state becomes issued
   * - `manual_approval_needed`: the invoice will be sent to the customer, the invoice state becomes issued
   */
  @post
  @route("/approve")
  @summary("Send the invoice to the customer")
  @operationId("approveInvoiceAction")
  @extension("x-internal", true)
  approveAction(
    @path
    invoiceId: ULID,
  ): Invoices.Invoice | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Void an invoice
   *
   * Only invoices that have been alread issued can be voided.
   *
   * Voiding an invoice will mark it as voided, the user can specify how to handle the voided line items.
   */
  @post
  @route("/void")
  @summary("Void an invoice")
  @operationId("voidInvoiceAction")
  @extension("x-internal", true)
  voidInvoiceAction(
    @path
    invoiceId: ULID,

    @body
    request: VoidInvoiceActionInput,
  ): Invoices.Invoice | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Advance the invoice's state to the next status.
   *
   * The call doesn't "approve the invoice", it only advances the invoice to the next status if the transition would be automatic.
   *
   * The action can be called when the invoice's statusDetails' actions field contain the "advance" action.
   */
  @post
  @route("/advance")
  @summary("Advance the invoice's state to the next status")
  @operationId("advanceInvoiceAction")
  @extension("x-internal", true)
  advanceAction(
    @path
    invoiceId: ULID,
  ): Invoices.Invoice | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Retry advancing the invoice after a failed attempt.
   *
   * The action can be called when the invoice's statusDetails' actions field contain the "retry" action.
   */
  @post
  @route("/retry")
  @summary("Retry advancing the invoice after a failed attempt.")
  @operationId("retryInvoiceAction")
  @extension("x-internal", true)
  retryAction(
    @path
    invoiceId: ULID,
  ): Invoices.Invoice | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  // Line management

  /**
   * Get an invoice line
   */
  @get
  @route("/lines/{lineId}")
  @summary("Get an invoice line")
  @operationId("getInvoiceLine")
  @extension("x-internal", true)
  getInvoiceLine(
    @path invoiceId: ULID,
    @path lineId: ULID,
  ): Invoices.InvoiceLine | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Delete an invoice line
   */
  @delete
  @route("/lines/{lineId}")
  @summary("Delete an invoice line")
  @operationId("deleteInvoiceLine")
  @extension("x-internal", true)
  deleteInvoiceLine(@path invoiceId: ULID, @path lineId: ULID): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Update an invoice line
   */
  @post
  @route("/lines/{lineId}")
  @summary("Update an invoice line")
  @operationId("updateInvoiceLine")
  @extension("x-internal", true)
  updateInvoiceLine(
    @path invoiceId: ULID,
    @path lineId: ULID,

    @body
    request: Invoices.InvoiceLineReplaceUpdate,
  ): Invoices.InvoiceLine | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * TODOs:
   * - min/max spend support
   * - discounts
   * - tax support (detailed)
   */
}

/**
 * InvoiceOrderBy specifies the ordering options for invoice listing.
 */
@friendlyName("InvoiceOrderBy")
enum InvoiceOrderBy {
  customerName: "customer.name",
  issuedAt: "issuedAt",
  status: "status",
  createdAt: "createdAt",
  updatedAt: "updatedAt",
}

/**
 * Request to void an invoice
 */
@friendlyName("VoidInvoiceActionInput")
model VoidInvoiceActionInput {
  /**
   * The action to take on the voided line items.
   */
  @visibility(Lifecycle.Create)
  action: VoidInvoiceAction;

  /**
   * The reason for voiding the invoice.
   */
  @visibility(Lifecycle.Create)
  reason: string;

  /**
   * Per line item overrides for the action.
   *
   * If not specified, the `action` will be applied to all line items.
   */
  @visibility(Lifecycle.Create)
  overrides?: VoidInvoiceActionLineOverride[] | null;
}

/**
 * InvoiceVoidAction describes how to handle the voided line items.
 */
@friendlyName("VoidInvoiceAction")
model VoidInvoiceAction {
  /**
   * How much of the total line items to be voided? (e.g. 100% means all charges are voided)
   */
  @visibility(Lifecycle.Create)
  percentage: Percentage;

  action: VoidInvoiceLineAction;
}

/**
 * VoidInvoiceLineActionType describes how to handle the voidied line item in the invoice.
 */
@friendlyName("VoidInvoiceLineActionType")
enum VoidInvoiceLineActionType {
  /**
   * The line items will never be charged for again
   */
  discard: "discard",

  /**
   * Queue the line items into the pending state, they will be included in the next invoice. (We want to generate an invoice right now)
   */
  pending: "pending",
}

/**
 * VoidInvoiceLineAction describes how to handle a specific line item in the invoice when voiding.
 */
@friendlyName("VoidInvoiceLineAction")
@discriminator("type")
union VoidInvoiceLineAction {
  discard: VoidInvoiceLineDiscardAction,
  pending: VoidInvoiceLinePendingAction,
}

/**
 * VoidInvoiceLineDiscardAction describes how to handle the voidied line item in the invoice.
 */
@friendlyName("VoidInvoiceLineDiscardAction")
model VoidInvoiceLineDiscardAction {
  type: VoidInvoiceLineActionType.discard;
}

/**
 * VoidInvoiceLinePendingAction describes how to handle the voidied line item in the invoice.
 */
@friendlyName("VoidInvoiceLinePendingAction")
model VoidInvoiceLinePendingAction {
  type: VoidInvoiceLineActionType.pending;

  /**
   * The time at which the line item should be invoiced again.
   *
   * If not provided, the line item will be re-invoiced now.
   */
  @visibility(Lifecycle.Create)
  nextInvoiceAt?: DateTime;
}

/**
 * VoidInvoiceLineOverride describes how to handle a specific line item in the invoice when voiding.
 */
@friendlyName("VoidInvoiceActionLineOverride")
model VoidInvoiceActionLineOverride {
  /**
   * The line item ID to override.
   */
  @visibility(Lifecycle.Create)
  lineId: ULID;

  /**
   * The action to take on the line item.
   */
  @visibility(Lifecycle.Create)
  action: VoidInvoiceAction;
}

/**
 * Common parameters for listing invoices
 */
@friendlyName("InvoiceListParams")
model InvoiceListParams {
  /**
   * Filter by the invoice status.
   */
  @query
  statuses?: Invoices.InvoiceStatus[];

  /**
   * Filter by invoice extended statuses
   */
  @query
  extendedStatuses?: Invoices.InvoiceExtendedStatus[];

  /**
   * Filter by invoice creation time
   */
  @query
  issuedAfter?: DateTime;

  /**
   * Filter by invoice creation time
   */
  @query
  issuedBefore?: DateTime;

  /**
   * What parts of the list output to expand in listings
   */
  @query
  expand?: InvoiceExpand[];

  /**
   * Filter by customer ID
   */
  @query
  customers?: ULID[];
}

/**
 * InvoiceExpand specifies the parts of the invoice to expand in the list output.
 */
@friendlyName("InvoiceExpand")
@extension("x-enum-varnames", ["all", "lines", "preceding", "workflowApps"])
enum InvoiceExpand {
  all: "*",
  lines: "lines",
  preceding: "preceding",
  workflowApps: "workflow.apps",
}

/**
 * BillingInvoiceActionInput is the input for creating an invoice.
 *
 * Invoice creation is always based on already pending line items created by the billingCreateLineByCustomer
 * operation. Empty invoices are not allowed.
 */
@friendlyName("InvoicePendingLinesActionInput")
model InvoicePendingLinesActionInput {
  /**
   * The pending line items to include in the invoice, if not provided:
   * - all line items that have invoice_at < asOf will be included
   * - all usage based line items will be included up to asOf, new usage-based line items will be staged for the rest
   *   of the billing cycle
   */
  @visibility(Lifecycle.Create)
  includePendingLines?: ULID[];

  /**
   * The time as of which the invoice is created.
   *
   * If not provided, the current time is used.
   */
  @visibility(Lifecycle.Create)
  asOf?: DateTime;

  /**
   * The customer ID for which to create the invoice.
   */
  @visibility(Lifecycle.Create)
  customerId: ULID;
}
