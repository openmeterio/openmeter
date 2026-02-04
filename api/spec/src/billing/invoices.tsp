using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;

/**
 * InvoicesEndpoints is a collection of endpoints that allow invoice operations without prior
 * knowledge of the invoice ID.
 */
@route("/api/v1/billing/invoices")
@tag("Billing")
@friendlyName("Invoices")
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
  invoicePendingLinesAction(
    @body
    request: InvoicePendingLinesActionInput,
  ): {
    @statusCode _: 201;

    @body
    body: Invoice[];
  } | CommonErrors;

  /**
   * List invoices based on the specified filters.
   *
   * The expand option can be used to include additional information (besides the invoice header and totals)
   * in the response. For example by adding the expand=lines option the invoice lines will be included in the response.
   *
   * Gathering invoices will always show the current usage calculated on the fly.
   */
  @get
  @summary("List invoices")
  @operationId("listInvoices")
  list(
    ...InvoiceListParams,
    ...QueryPagination,
    ...QueryOrdering<InvoiceOrderBy>,
  ): PaginatedResponse<Invoice> | CommonErrors;
}

/**
 * CustomerInvoice is a collection of endpoints that allow operations on a specific invoice.
 */
@route("/api/v1/billing/invoices/{invoiceId}")
@tag("Billing")
@friendlyName("Invoice")
interface InvoiceEndpoints {
  /**
   * Get an invoice by ID.
   *
   * Gathering invoices will always show the current usage calculated on the fly.
   */
  @get
  @summary("Get an invoice")
  @operationId("getInvoice")
  getInvoice(
    @path
    invoiceId: ULID,

    @query(#{ explode: true })
    expand?: InvoiceExpand[] = #[InvoiceExpand.lines],

    @query
    includeDeletedLines?: boolean = false,
  ): Invoice | NotFoundError | CommonErrors;

  /**
   * Delete an invoice
   *
   * Only invoices that are in the draft (or earlier) status can be deleted.
   *
   * Invoices that are post finalization can only be voided.
   */
  @delete
  @summary("Delete an invoice")
  @operationId("deleteInvoice")
  deleteInvoice(
    @path
    invoiceId: ULID,
  ): {
    @statusCode _: 204;
  } | NotFoundError | CommonErrors;

  /**
   * Update an invoice
   *
   * Only invoices in draft or earlier status can be updated.
   */
  @put
  @summary("Update an invoice")
  @operationId("updateInvoice")
  updateInvoice(
    @path
    invoiceId: ULID,

    @body
    request: InvoiceReplaceUpdate,
  ): Invoice | NotFoundError | CommonErrors;

  /**
   * Recalculate an invoice's tax amounts (using the app set in the customer's billing profile)
   *
   * Note: charges might apply, depending on the tax provider.
   */
  @post
  @route("/taxes/recalculate")
  @summary("Recalculate an invoice's tax amounts")
  @operationId("recalculateInvoiceTaxAction")
  recalculateTaxAction(
    @path
    invoiceId: ULID,
  ): Invoice | NotFoundError | CommonErrors;

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
  approveAction(
    @path
    invoiceId: ULID,
  ): Invoice | NotFoundError | CommonErrors;

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
  voidInvoiceAction(
    @path
    invoiceId: ULID,

    @body
    request: VoidInvoiceActionInput,
  ): Invoice | NotFoundError | CommonErrors;

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
  advanceAction(
    @path
    invoiceId: ULID,
  ): Invoice | NotFoundError | CommonErrors;

  /**
   * Retry advancing the invoice after a failed attempt.
   *
   * The action can be called when the invoice's statusDetails' actions field contain the "retry" action.
   */
  @post
  @route("/retry")
  @summary("Retry advancing the invoice after a failed attempt.")
  @operationId("retryInvoiceAction")
  retryAction(
    @path
    invoiceId: ULID,
  ): Invoice | NotFoundError | CommonErrors;

  /**
   * Snapshot quantities for usage based line items.
   *
   * This call will snapshot the quantities for all usage based line items in the invoice.
   *
   * This call is only valid in `draft.waiting_for_collection` status, where the collection period
   * can be skipped using this action.
   */
  @post
  @route("/snapshot-quantities")
  @summary("Snapshot quantities for usage based line items")
  @operationId("snapshotQuantitiesInvoiceAction")
  snapshotQuantitiesAction(
    @path
    invoiceId: ULID,
  ): Invoice | NotFoundError | CommonErrors;
}

@route("/api/v1/billing/customers/{customerId}/invoices")
@tag("Billing")
@friendlyName("CustomerInvoice")
interface CustomerInvoiceEndpoints {
  /**
   * Simulate an invoice for a customer.
   *
   * This call will simulate an invoice for a customer based on the pending line items.
   *
   * The call will return the total amount of the invoice and the line items that will be included in the invoice.
   */
  @post
  @summary("Simulate an invoice for a customer")
  @route("/simulate")
  @operationId("simulateInvoice")
  simulateInvoice(
    @path
    customerId: ULID,

    @body
    request: InvoiceSimulationInput,
  ): Invoice | CommonErrors;

  // TODO[later]: Let's add a @get to list upcoming charges without the gathering invoices
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
  @summary("Create pending line items")
  @route("/pending-lines")
  @operationId("createPendingInvoiceLine")
  createPendingInvoiceLine(
    @path
    customerId: ULID,

    @body
    request: InvoicePendingLineCreateInput,
  ): {
    @statusCode _: 201;
    @body body: InvoicePendingLineCreateResponse;
  } | CommonErrors;
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
  periodStart: "periodStart",
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

  /**
   * The action to take on the line items.
   */
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
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union VoidInvoiceLineAction {
  discard: VoidInvoiceLineDiscardAction,
  pending: VoidInvoiceLinePendingAction,
}

/**
 * VoidInvoiceLineDiscardAction describes how to handle the voidied line item in the invoice.
 */
@friendlyName("VoidInvoiceLineDiscardAction")
model VoidInvoiceLineDiscardAction {
  /**
   * The action to take on the line item.
   */
  type: VoidInvoiceLineActionType.discard;
}

/**
 * VoidInvoiceLinePendingAction describes how to handle the voidied line item in the invoice.
 */
@friendlyName("VoidInvoiceLinePendingAction")
model VoidInvoiceLinePendingAction {
  /**
   * The action to take on the line item.
   */
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
  @query(#{ explode: true })
  statuses?: InvoiceStatus[];

  /**
   * Filter by invoice extended statuses
   */
  @query(#{ explode: true })
  extendedStatuses?: InvoiceExtendedStatus[];

  /**
   * Filter by invoice issued time.
   * Inclusive.
   */
  @query(#{ explode: true })
  issuedAfter?: DateTime;

  /**
   * Filter by invoice issued time.
   * Inclusive.
   */
  @query(#{ explode: true })
  issuedBefore?: DateTime;

  /**
   * Filter by period start time.
   * Inclusive.
   */
  @query(#{ explode: true })
  periodStartAfter?: DateTime;

  /**
   * Filter by period start time.
   * Inclusive.
   */
  @query(#{ explode: true })
  periodStartBefore?: DateTime;

  /**
   * Filter by invoice created time.
   * Inclusive.
   */
  @query(#{ explode: true })
  createdAfter?: DateTime;

  /**
   * Filter by invoice created time.
   * Inclusive.
   */
  @query(#{ explode: true })
  createdBefore?: DateTime;

  /**
   * What parts of the list output to expand in listings
   */
  @query(#{ explode: true })
  expand?: InvoiceExpand[];

  /**
   * Filter by customer ID
   */
  @query(#{ explode: true })
  customers?: ULID[];

  /**
   * Include deleted invoices
   */
  @query
  includeDeleted?: boolean;
}

/**
 * InvoiceExpand specifies the parts of the invoice to expand in the list output.
 */
@friendlyName("InvoiceExpand")
enum InvoiceExpand {
  lines: "lines",
  preceding: "preceding",

  /**
   * @deprecated We are always expanding the workflow apps.
   */
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
   * Filters to apply when creating the invoice.
   */
  @visibility(Lifecycle.Create)
  filters?: InvoicePendingLinesActionFiltersInput;

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

  /**
   * Override the progressive billing setting of the customer.
   *
   * Can be used to disable/enable progressive billing in case the business logic
   * requires it, if not provided the billing profile's progressive billing setting will be used.
   */
  @visibility(Lifecycle.Create)
  progressiveBillingOverride?: boolean;
}

/**
 * InvoicePendingLinesActionFiltersInput specifies which lines to include in the invoice.
 */
@friendlyName("InvoicePendingLinesActionFiltersInput")
model InvoicePendingLinesActionFiltersInput {
  /**
   * The pending line items to include in the invoice, if not provided:
   * - all line items that have invoice_at < asOf will be included
   * - [progressive billing only] all usage based line items will be included up to asOf, new
   *   usage-based line items will be staged for the rest of the billing cycle
   *
   * All lineIDs present in the list, must exists and must be invoicable as of asOf, or the action will fail.
   */
  @visibility(Lifecycle.Create)
  lineIds?: ULID[];
}

/**
 * InvoiceSimulationInput is the input for simulating an invoice.
 */
@friendlyName("InvoiceSimulationInput")
model InvoiceSimulationInput {
  /**
   * The number of the invoice.
   */
  @visibility(Lifecycle.Create)
  number?: InvoiceNumber;

  /**
   * Currency for all invoice line items.
   *
   * Multi currency invoices are not supported yet.
   */
  @visibility(Lifecycle.Create)
  currency: CurrencyCode;

  /**
   * Lines to be included in the generated invoice.
   */
  @visibility(Lifecycle.Create)
  lines: InvoiceSimulationLine[];
}

/**
 * InvoiceSimulationLine represents a usage-based line item that can be input to the simulation endpoint.
 */
@friendlyName("InvoiceSimulationLine")
model InvoiceSimulationLine {
  ...TypeSpec.Rest.Resource.ResourceCreateModel<OmitProperties<
    InvoiceLine,
    "invoice" | "currency" | "children"
  >>;

  /**
   * The quantity of the item being sold.
   */
  @visibility(Lifecycle.Create)
  quantity: Numeric;

  /**
   * The quantity of the item used before this line's period, if the line is billed progressively.
   */
  @visibility(Lifecycle.Create)
  preLinePeriodQuantity?: Numeric;

  /**
   * ID of the line. If not specified it will be auto-generated.
   *
   * When discounts are specified, this must be provided, so that the discount can reference it.
   */
  @visibility(Lifecycle.Create)
  id?: ULID;
}

/**
 * InvoiceReplaceUpdate represents the update model for an invoice.
 */
@friendlyName("InvoiceReplaceUpdate")
model InvoiceReplaceUpdate {
  ...TypeSpec.Rest.Resource.ResourceReplaceModel<OmitProperties<
    Invoice,
    "lines" | "supplier" | "customer" | "discounts" | "workflow" | "draftUntil"
  >>;

  /**
   * The supplier of the lines included in the invoice.
   */
  @visibility(Lifecycle.Update)
  supplier: TypeSpec.Rest.Resource.ResourceReplaceModel<BillingParty>;

  /**
   * The customer the invoice is sent to.
   */
  @visibility(Lifecycle.Update)
  customer: TypeSpec.Rest.Resource.ResourceReplaceModel<BillingParty>;

  /**
   * The lines included in the invoice.
   */
  @visibility(Lifecycle.Update)
  lines: InvoiceLineReplaceUpdate[];

  /**
   * The workflow settings for the invoice.
   */
  @visibility(Lifecycle.Update)
  workflow: InvoiceWorkflowReplaceUpdate;
}

/**
 * InvoiceWorkflowReplaceUpdate represents the update model for an invoice workflow.
 *
 * Fields that are immutable a re removed from the model. This is based on InvoiceWorkflowSettings.
 */
@friendlyName("InvoiceWorkflowReplaceUpdate")
model InvoiceWorkflowReplaceUpdate {
  /**
   * The workflow used for this invoice.
   */
  workflow: InvoiceWorkflowSettingsReplaceUpdate;
}

/**
 * Mutable workflow settings for an invoice.
 *
 * Other fields on the invoice's workflow are not mutable, they serve as a history of the invoice's workflow
 * at creation time.
 */
@friendlyName("InvoiceWorkflowSettingsReplaceUpdate")
model InvoiceWorkflowSettingsReplaceUpdate {
  /**
   * The invoicing settings for this workflow
   */
  @visibility(Lifecycle.Update)
  invoicing: InvoiceWorkflowInvoicingSettingsReplaceUpdate;

  /**
   * The payment settings for this workflow
   */
  @visibility(Lifecycle.Update)
  payment: BillingWorkflowPaymentSettings;
}

/**
 * InvoiceWorkflowInvoicingSettingsReplaceUpdate represents the update model for the invoicing settings of an invoice workflow.
 */
@friendlyName("InvoiceWorkflowInvoicingSettingsReplaceUpdate")
model InvoiceWorkflowInvoicingSettingsReplaceUpdate {
  ...TypeSpec.Rest.Resource.ResourceReplaceModel<OmitProperties<
    BillingWorkflowInvoicingSettings,
    "progressiveBilling"
  >>;
}

/**
 * InvoicePendingLineCreate represents the create model for a pending invoice line.
 */
@friendlyName("InvoicePendingLineCreateInput")
model InvoicePendingLineCreateInput {
  /**
   * The currency of the lines to be created.
   */
  @visibility(Lifecycle.Create)
  currency: CurrencyCode;

  /**
   * The lines to be created.
   */
  @visibility(Lifecycle.Create)
  @minItems(1)
  lines: InvoicePendingLineCreate[];
}

/**
 * InvoicePendingLineCreateResponse represents the response from the create pending line endpoint.
 */
@friendlyName("InvoicePendingLineCreateResponse")
model InvoicePendingLineCreateResponse {
  /**
   * The lines that were created.
   */
  @visibility(Lifecycle.Read)
  lines: InvoiceLine[];

  // TODO: For the V3 api let's not return the invoice
  /**
   * The invoice containing the created lines.
   */
  @visibility(Lifecycle.Read)
  invoice: Invoice;

  /**
   * Whether the invoice was newly created.
   */
  @visibility(Lifecycle.Read)
  isInvoiceNew: boolean;
}
