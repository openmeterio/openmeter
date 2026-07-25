import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "../billing/profile.tsp";
import "./invoice.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Invoices;

/**
 * Filter options for listing invoices.
 */
@friendlyName("ListInvoicesParamsFilter")
model ListInvoicesParamsFilter {
  /**
   * Filter by invoice status.
   */
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  status?: Common.StringFieldFilterExact;

  /**
   * Filter by customer ID.
   */
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  customer_id?: Common.ULIDFieldFilter;

  /**
   * Filter by the time the invoice was issued.
   */
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  issued_at?: Common.DateTimeFieldFilter;

  /**
   * Filter by service period start.
   */
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  service_period_start?: Common.DateTimeFieldFilter;

  /**
   * Filter by invoice creation time.
   */
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  created_at?: Common.DateTimeFieldFilter;
}

/**
 * Operations for managing billing invoices.
 */
interface InvoicesOperations {
  /**
   * List billing invoices.
   *
   * Returns a page of invoices. Gathering invoices are never included. Use `filter`
   * to narrow by status, customer, dates, or service period start. Use `sort` to
   * control ordering.
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  @get
  @operationId("list-invoices")
  @summary("List billing invoices")
  list(
    ...Common.PagePaginationQuery,

    /**
     * Sort invoices returned in the response. Supported sort attributes:
     *
     * - `issued_at`
     * - `created_at` (default)
     * - `service_period_start`
     *
     * The `asc` suffix is optional as the default sort order is ascending. The `desc`
     * suffix is used to specify a descending order.
     */
    @query(#{ name: "sort" })
    sort?: Common.SortQuery,

    /**
     * Filter invoices returned in the response.
     *
     * Examples:
     *
     * - `filter[status][oeq]=draft,issued`
     * - `filter[customer_id]=01KPDB8K...`
     * - `filter[issued_at][gte]=2024-01-01T00:00:00Z`
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListInvoicesParamsFilter,
  ): Shared.PagePaginatedResponse<Invoice> | Common.ErrorResponses;

  /**
   * Get a billing invoice by ID.
   *
   * Returns the full invoice resource including line items, status details, totals,
   * and workflow configuration snapshot.
   */
  @get
  @operationId("get-invoice")
  @summary("Get a billing invoice")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  get(@path invoiceId: Shared.ULID):
    | Shared.GetResponse<Invoice>
    | Common.NotFound
    | Common.ErrorResponses;

  /**
   * Update a billing invoice.
   *
   * Only the mutable fields of the invoice can be edited: description, labels,
   * supplier, customer, workflow settings, and top-level lines. Top-level lines are
   * matched by `id`; lines without an `id` are created, and existing lines omitted
   * from `lines` are deleted. Detailed (child) lines are always computed and cannot
   * be edited directly. Only invoices in draft status can be updated.
   */
  @put
  @operationId("update-invoice")
  @summary("Update a billing invoice")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  update(
    @path invoiceId: Shared.ULID,

    @body
    invoice: UpdateInvoiceRequest,
  ): Shared.UpdateResponse<Invoice> | Common.NotFound | Common.ErrorResponses;

  /**
   * Delete a billing invoice.
   *
   * Only standard invoices in draft status can be deleted. Deleting an invoice will
   * also delete all associated line items and workflow configuration.
   */
  @delete
  @operationId("delete-invoice")
  @summary("Delete a billing invoice")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  delete(@path invoiceId: Shared.ULID):
    | Shared.DeleteResponse
    | Common.NotFound
    | Common.ErrorResponses;

  /**
   * Advance a billing invoice.
   *
   * Advances the invoice to the next workflow state. The next state is determined by
   * the invoice's current status and workflow configuration. Only invoices in draft
   * or issued status can be advanced.
   *
   * @returns The updated invoice after advancing to the next state.
   */
  @post
  @route("/{invoiceId}/advance")
  @operationId("advance-invoice")
  @summary("Advance billing invoice's next status")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  advance(@path invoiceId: Shared.ULID):
    | Invoice
    | Common.NotFound
    | Common.ErrorResponses;

  /**
   * Approve a billing invoice.
   *
   * This call instantly sends the invoice to the customer using the configured
   * billing profile app.
   *
   * This call is valid in two invoice statuses:
   *
   * - draft: the invoice will be sent to the customer, the invoice state becomes
   *   issued
   * - manual_approval_needed: the invoice will be sent to the customer, the invoice
   *   state becomes issued
   *
   * @returns The updated invoice after sending to the customer.
   */
  @post
  @route("/{invoiceId}/approve")
  @operationId("approve-invoice")
  @summary("Send the invoice to the customer")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  approve(@path invoiceId: Shared.ULID):
    | Invoice
    | Common.NotFound
    | Common.ErrorResponses;

  /**
   * Retry sending a billing invoice.
   *
   * Retry advancing the invoice after a failed attempt.
   *
   * The action can be called when the invoice's statusDetails' actions field contain
   * the "retry" action.
   *
   * @returns The updated invoice after retrying the action.
   */
  @post
  @route("/{invoiceId}/retry")
  @operationId("retry-invoice")
  @summary("Retry advancing the invoice after a failed attempt")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  retry(@path invoiceId: Shared.ULID):
    | Invoice
    | Common.NotFound
    | Common.ErrorResponses;

  /**
   * Snapshot quantities for usage-based line items.
   *
   * This call will snapshot the quantities for all usage based line items in the
   * invoice.
   *
   * This call is only valid in draft.waiting_for_collection status, where the
   * collection period can be skipped using this action.
   *
   * @returns The updated invoice with the snapshot quantities for usage based line
   * items.
   */
  @post
  @route("/{invoiceId}/snapshot-quantities")
  @operationId("snapshot-quantities-invoice")
  @summary("Snapshot quantities for usage based line items")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  snapshotQuantities(@path invoiceId: Shared.ULID):
    | Invoice
    | Common.NotFound
    | Common.ErrorResponses;
}
