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
}
