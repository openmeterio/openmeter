import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./meter.tsp";
import "./query.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Meters;

/**
 * Filter options for listing meters.
 */
@friendlyName("ListMetersParamsFilter")
model ListMetersParamsFilter {
  /**
   * Filter meters by key.
   */
  key?: Common.StringFieldFilter;

  /**
   * Filter meters by name.
   */
  name?: Common.StringFieldFilter;
}

interface MetersOperations {
  /**
   * Create a meter.
   */
  @post
  @operationId("create-meter")
  @summary("Create meter")
  create(
    @body meter: Shared.CreateRequest<Meter>,
  ): Shared.CreateResponse<Meter> | Common.ErrorResponses;

  /**
   * Get a meter by ID.
   */
  @get
  @operationId("get-meter")
  @summary("Get meter")
  get(
    @path meterId: Shared.ULID,
  ): Shared.GetResponse<Meter> | Common.NotFound | Common.ErrorResponses;

  /**
   * List meters.
   */
  @get
  @operationId("list-meters")
  @summary("List meters")
  list(
    ...Common.PagePaginationQuery,

    /**
     * Filter meters returned in the response.
     *
     * To filter meters by key add the following query param: filter[key]=my-meter-key
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListMetersParamsFilter,
  ): Shared.PagePaginatedResponse<Meter> | Common.ErrorResponses;

  /**
   * Update a meter.
   */
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @put
  @operationId("update-meter")
  @summary("Update meter")
  update(
    @path meterId: Shared.ULID,
    @body meter: Shared.UpdateRequest<Meter>,
  ): Shared.UpdateResponse<Meter> | Common.NotFound | Common.ErrorResponses;

  /**
   * Delete a meter.
   */
  @delete
  @operationId("delete-meter")
  @summary("Delete meter")
  delete(
    @path meterId: Shared.ULID,
  ): Shared.DeleteResponse | Common.NotFound | Common.ErrorResponses;

  /**
   * Query a meter for usage.
   */
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @post
  @operationId("query-meter")
  @summary("Query meter")
  query(
    @path meterId: Shared.ULID,
    @body request: MeterQueryRequest,
  ): MeterQueryResult | Common.NotFound | Common.ErrorResponses;
}
