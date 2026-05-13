import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../common/properties.tsp";
import "../shared/index.tsp";
import "./period.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Accounting;

/**
 * Filter options for listing accounting periods.
 */
@friendlyName("ListAccountingPeriodsParamsFilter")
model ListAccountingPeriodsParamsFilter {
  /**
   * Filter periods by lifecycle status. Use `filter[status][oeq]=open,closing` to
   * fetch the currently-active periods.
   */
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  status?: Common.StringFieldFilterExact;

  /**
   * Filter periods by `start_at`.
   */
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  start_at?: Common.DateTimeFieldFilter;

  /**
   * Filter periods by `end_at`. Open periods (no `end_at`) are excluded when this
   * filter is applied.
   */
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  end_at?: Common.DateTimeFieldFilter;

  /**
   * Filter periods by labels.
   */
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  labels?: Common.LabelsFieldFilter;
}

/**
 * Request body for initiating a close on the currently-open accounting period.
 */
@friendlyName("CloseAccountingPeriodRequest")
@summary("Close accounting period request")
model CloseAccountingPeriodRequest {
  /**
   * The intended end of the period being closed (exclusive). Must be strictly
   * greater than the open period's `start_at` and not in the future.
   *
   * Becomes the closing period's `end_at` and the newly-opened next period's
   * `start_at`.
   */
  @summary("End of period")
  end_at: Shared.DateTime;

  /**
   * Optional labels applied to the closing period.
   */
  @summary("Labels")
  labels?: Common.Labels;

  /**
   * Optional display name for the period (e.g. "2026-Q2"). Auto-generated if
   * omitted.
   */
  @summary("Name")
  @minLength(1)
  @maxLength(256)
  name?: string;

  /**
   * Optional description.
   */
  @summary("Description")
  @maxLength(1024)
  description?: string;
}

interface AccountingPeriodsOperations {
  /**
   * Close the currently-open accounting period.
   *
   * Atomically, under a namespace-period advisory lock:
   *
   * 1. Freezes membership on the currently-open period `P`: sets `P.end_at`,
   *    transitions `P.status` to `closing`, and records `P.closing_started_at`.
   * 2. Opens a new period `N` with `N.start_at = P.end_at` and `N.status = open`.
   *
   * The transition `closing → closed` (balance snapshot computation, close-time
   * validations, and seal hash commit) is performed by the close pipeline; poll
   * `GET .../{periodId}` to observe the final `closed` state, `closed_at`,
   * `sealed_at`, and seal fields.
   *
   * Late-arriving transactions whose `booked_at` falls inside the now-closed period
   * are posted to `N` with an `adjusts_period_id` backpointer to `P`; they do not
   * modify `P`.
   *
   * Returns the period transitioned to `closing`. The newly-opened next period can
   * be retrieved by listing with `filter[status]=open`.
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @post
  @route("/close")
  @operationId("close-accounting-period")
  @summary("Close the open accounting period")
  close(
    @body body: CloseAccountingPeriodRequest,
  ): Shared.CreateResponse<AccountingPeriod> | Common.ErrorResponses;

  /**
   * Get an accounting period by id.
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @get
  @operationId("get-accounting-period")
  @summary("Get an accounting period")
  get(
    @path periodId: Shared.ULID,
  ): Shared.GetResponse<AccountingPeriod> | Common.NotFound | Common.ErrorResponses;

  /**
   * List accounting periods.
   *
   * Returns the namespace's periods in reverse-chronological order by `start_at` by
   * default (most recent first).
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @get
  @operationId("list-accounting-periods")
  @summary("List accounting periods")
  list(
    ...Common.PagePaginationQuery,

    /**
     * Sort periods returned in the response. Supported sort attributes are:
     *
     * - `start_at` (default, descending)
     * - `created_at`
     * - `id`
     */
    @query(#{ name: "sort" })
    sort?: Common.SortQuery,

    /**
     * Filter periods returned in the response.
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListAccountingPeriodsParamsFilter,
  ): Shared.PagePaginatedResponse<AccountingPeriod> | Common.ErrorResponses;
}
