using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.Billing;

/**
 * InvoicesAllCustomers is a collection of endpoints that allow invoice operations without prior
 * knowledge of the customer ID.
 */
@route("/api/v1/billing/invoices")
@tag("Billing (Experimental)")
interface InvoicesAllCustomers {
  /**
   * List invoices
   */
  @get
  @summary("List invoices")
  @operationId("billingListInvoices")
  listInvoices(
    @query
    @summary("Filter by the customer ID or key")
    customers?: Array<ULIDOrKey>,

    ...InvoiceListParams,
    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryLimitOffset,
    ...OpenMeter.QueryOrdering<InvoiceOrderBy>,
  ): OpenMeter.PaginatedResponse<Invoice.Bill.InvoiceSummary> | OpenMeter.CommonErrors;
}

/**
 * CustomerAllInvoices is a collection of endpoints that allow invoice operations without prior
 * knowledge of the invoice ID.
 */
@route("/api/v1/billing/invoices/{customerIdOrKey}")
@tag("Billing (Experimental)")
interface CustomerAllInvoices {
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
  @operationId("billingCreateInvoice")
  create(
    @path
    customerIdOrKey: ULIDOrKey,

    @body
    request: InvoiceCreateInput,
  ): {
    @statusCode _: 201;
    @body body: Invoice.Bill.Invoice;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * List invoices for a specific customer
   */
  @get
  @summary("List invoices")
  @operationId("billingListInvoicesByCustomer")
  listInvoicesByCustomer(
    @path customerIdOrKey: ULIDOrKey,
    ...InvoiceListParams,
    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryLimitOffset,
    ...OpenMeter.QueryOrdering<InvoiceOrderBy>,
  ): OpenMeter.PaginatedResponse<Invoice.Bill.InvoiceSummary> | OpenMeter.CommonErrors;

  /**
   * Create a new (pending line item) charge.
   *
   * This call is used to create a new pending charge for the customer without explicitly
   * assigning it to an invoice.
   *
   * The item will be either allocated to an existing invoice in gathering state or a new invoice is
   * created for the item.
   *
   * A new invoice will be created if:
   * - there is no invoice in gathering state
   * - the currency of the item doesn't match the currency of any invoices in gathering state
   */
  @post
  @route("/items")
  @summary("Create a new pending charge")
  @operationId("billingCreatePendingChargeByCustomer")
  createPendingCharge(@path customerIdOrKey: ULIDOrKey, @body request: Invoice.Bill.InvoiceLine): {
    @statusCode _: 201;
    @body body: CreatePendingChargeResponse;
  } | OpenMeter.CommonErrors;
}

@route("/api/v1/billing/invoices/{customerIdOrKey}/invoices/{invoiceId}")
@tag("Billing (Experimental)")
interface CustomerInvoices {
  /**
   * Get an invoice by ID.
   */
  @get
  @summary("Get an invoice")
  @operationId("billingGetInvoiceByID")
  getInvoiceByID(
    @path
    customerIdOrKey: ULIDOrKey,

    @path
    invoiceId: ULID,
  ): Invoice.Bill.Invoice | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Delete an invoice
   *
   * Only invoices that are in the draft (or earlier) status can be deleted.
   */
  @delete
  @summary("Delete an invoice")
  @operationId("billingDeleteInvoice")
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
  @route("/tax/recalculate")
  @summary("Recalculate an invoice's tax amounts")
  @operationId("billingRecalculateInvoiceTax")
  recalculateTax(
    @path
    customerIdOrKey: ULIDOrKey,

    @path
    invoiceId: ULID,
  ): {
    @statusCode _: 200;
    @body body: Invoice.Bill.Invoice;
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
  @route("/approve")
  @summary("Send the invoice to the customer")
  @operationId("billingApproveInvoice")
  approve(
    @path
    customerIdOrKey: ULIDOrKey,

    @path
    invoiceId: ULID,
  ): {
    @statusCode _: 200;
    @body body: Invoice.Bill.Invoice;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Void an invoice
   *
   * Only invoices that have been alread issued can be voided.
   *
   * Voiding an invoice will mark it as voided, the user can specify how to handle the voided items.
   */
  @post
  @route("/void")
  @summary("Void an invoice")
  @operationId("billingVoidInvoice")
  voidInvoice(
    @path
    customerIdOrKey: ULIDOrKey,

    @path
    invoiceId: ULID,

    @body
    request: VoidInvoiceInput,
  ): {
    @statusCode _: 200;
    @body body: Invoice.Bill.Invoice;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Update invoice workflow
   *
   * This call can be used to update the invoice workflow, only those settings are adjustable
   * that are not referring to past activities (such as collection).
   */
  @put
  @route("/workflow")
  @summary("Update invoice workflow")
  @operationId("billingUpdateInvoiceWorkflow")
  updateWorkflow(
    @path
    customerIdOrKey: ULIDOrKey,

    @path
    invoiceId: ULID,

    @body
    request: Invoice.Bill.OpenMeter.InvoiceWorkflowSettings,
  ): {
    @statusCode _: 200;
    @body body: Invoice.Bill.Invoice;
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
  @route("/workflow/advance")
  @summary("Advance the invoice's state to the next status")
  @operationId("billingInvoiceWorkflowAdvance")
  advanceWorkflow(
    @path
    customerIdOrKey: ULIDOrKey,

    @path
    invoiceId: ULID,
  ): {
    @statusCode _: 200;
    @body body: Invoice.Bill.Invoice;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  // Contact management

  /**
   * Update/override the supplier information for the invoice
   *
   * Can be called only on invoices in the draft status.
   */
  @put
  @route("/supplier")
  @summary("Update the supplier information for the invoice")
  @operationId("billingUpdateInvoiceSupplier")
  updateInvoiceSupplier(
    @path
    customerIdOrKey: ULIDOrKey,

    @path
    invoiceId: ULID,

    @body
    request: Invoice.Org.Party,
  ): {
    @statusCode _: 200;
    @body body: Invoice.Org.Party;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Update/override the customer information for the invoice
   *
   * Can be called only on invoices in the draft status.
   */
  @put
  @route("/customer")
  @summary("Update the customer information for the invoice")
  @operationId("billingUpdateInvoiceCustomer")
  updateCustomer(
    @path
    customerIdOrKey: ULIDOrKey,

    @path
    invoiceId: ULID,

    @body
    request: Invoice.Org.Party,
  ): {
    @statusCode _: 200;
    @body body: Invoice.Org.Party;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  // Line management

  /**
   * Delete an invoice line
   */
  @delete
  @route("/lines/{lineId}")
  @summary("Delete an invoice line")
  @operationId("billingDeleteInvoiceLine")
  deleteInvoiceLine(@path customerIdOrKey: ULIDOrKey, @path invoiceId: ULID, @path lineId: ULID): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Update an invoice line
   */
  @put
  @route("/lines/{lineId}")
  @summary("Update an invoice line")
  @operationId("billingUpdateInvoiceLine")
  updateInvoiceLine(
    @path customerIdOrKey: ULIDOrKey,
    @path invoiceId: ULID,
    @path lineId: ULID,
    @body request: Invoice.Bill.InvoiceLine,
  ): {
    @statusCode _: 200;
    @body body: Invoice.Bill.InvoiceLine;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

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
@friendlyName("BillingInvoiceOrderBy")
enum InvoiceOrderBy {
  customer_name: "customer_name",
  issued_at: "issued_at",
  status: "status",
  created: "created",
  updated: "updated",
}

/**
 * Response for creating a pending charge
 */
@friendlyName("BillingCreatePendingChargeResponse")
model CreatePendingChargeResponse {
  @summary("The created pending charge")
  item: Invoice.Bill.InvoiceLine;

  @summary("The invoice the item was added to")
  invoice: Invoice.Bill.InvoiceSummary;
}

/**
 * Request to void an invoice
 */
@friendlyName("BillingVoidInvoiceInput")
model VoidInvoiceInput {
  /**
   * The action to take on the voided items.
   */
  @summary("The action to take on the voided items")
  action: VoidInvoiceItemAction;

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
  itemOverrides: VoidInvoiceItemOverride[];
}

/**
 * InvoiceVoidAction describes how to handle the voided items.
 */
@friendlyName("BillingVoidInvoiceAction")
model VoidInvoiceAction {
  @summary("How much of the total items to be voided? (e.g. 100% means all charges are voided)")
  percentage: Percentage;

  @summary("How to handle the voided items, default: pending")
  action: VoidInvoiceItemAction;
}

@summary("VoidInvoiceItemAction describes how to handle the voidied item in the invoice.")
@friendlyName("BillingVoidInvoiceItemAction")
enum VoidInvoiceItemAction {
  @summary("The items will never be charged for again")
  discard: "discard",

  @summary("Queue the items into the pending state, they will be included in the next invoice. (We want to generate an invoice right now)")
  pending: "pending",

  @summary("Queue the items into the pending state, they will be included in the next invoice, but not in the current one")
  pending_next_cycle: "pending_next_cycle",
}

/**
 * VoidInvoiceItemOverride describes how to handle a specific item in the invoice when voiding.
 */
@friendlyName("BillingVoidInvoiceItemOverride")
model VoidInvoiceItemOverride {
  @summary("The item ID to override")
  itemId: ULID;

  @summary("The action to take on the item")
  action: VoidInvoiceAction;
}

/**
 * Common parameters for listing invoices
 */
@friendlyName("BillingInvoiceListParams")
model InvoiceListParams {
  @query
  @summary("Filter by the invoice status")
  statuses?: Array<Invoice.Bill.OpenMeter.InvoiceStatus>;

  @query
  @summary("Filter by invoice creation time")
  issuedAfter?: DateTime;

  @query
  @summary("Filter by invoice creation time")
  issuedBefore?: DateTime;
}

@friendlyName("BillingInvoiceCreateInput")
model InvoiceCreateInput {
  /**
   * The pending items to include in the invoice, if not provided all pending items will be included.
   */
  IncludePendingItems?: ULID[];
}
