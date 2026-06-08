import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./app.tsp";
import "./external_invoicing.tsp";
import "./stripe.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Apps;

/**
 * Filter options for listing apps.
 */
@friendlyName("ListAppsParamsFilter")
model ListAppsParamsFilter {
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  id?: Common.ULIDFieldFilter;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  name?: Common.StringFieldFilter;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  type?: Common.StringFieldFilterExact;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  status?: Common.StringFieldFilterExact;
}

interface AppsOperations {
  /**
   * List installed apps.
   */
  @get
  @operationId("list-apps")
  @summary("List apps")
  list(
    ...Common.PagePaginationQuery,

    /**
     * Sort apps returned in the response. Supported sort attributes are:
     *
     * - `id`
     * - `created_at` (default)
     *
     * The `asc` suffix is optional as the default sort order is ascending. The `desc`
     * suffix is used to specify a descending order.
     */
    @query(#{ name: "sort" })
    sort?: Common.SortQuery,

    /**
     * Filter apps returned in the response.
     *
     * To filter apps by name add the following query param: filter[name]=my-app
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListAppsParamsFilter,
  ): Shared.PagePaginatedResponse<App> | Common.ErrorResponses;

  /**
   * Get an installed app.
   */
  @get
  @operationId("get-app")
  @summary("Get app")
  get(@path appId: Shared.ULID):
    | Shared.GetResponse<App>
    | Common.NotFound
    | Common.ErrorResponses;
}
