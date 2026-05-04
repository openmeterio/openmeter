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
}

interface MetersQueryOperations {
  /**
   * Query a meter for usage.
   *
   * Set `Accept: application/json` (the default) to get a structured JSON response.
   * Set `Accept: text/csv` to download the same data as a CSV file suitable for
   * spreadsheets. The CSV columns, in order, are:
   *
   * `from, to, [subject,] [customer_id, customer_key, customer_name,] <dimensions...>, value`
   *
   * The `subject` column is emitted only when `subject` is in the query's
   * `group_by_dimensions`.
   * The three `customer_*` columns are emitted together only when `customer_id`
   * is in the query's `group_by_dimensions`.
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @post
  @operationId("query-meter")
  @summary("Query meter")
  @sharedRoute
  query(@path meterId: Shared.ULID, @body request: MeterQueryRequest): {
    @header contentType: "application/json";
    @body _: MeterQueryResult;
  } | Common.NotFound | Common.ErrorResponses;

  #suppress "@openmeter/api-spec-aip/operation-summary" "Avoid duplicating the summary in OpenAPI yaml"
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @opExample(
    #{
      returnType: #{
        contentType: "text/csv",
        _: """
          from,to,value
          2023-01-01T00:00:00Z,2023-01-01T00:01:00Z,12
          2023-01-01T00:01:00Z,2023-01-01T00:02:00Z,20
          2023-01-01T00:02:00Z,2023-01-01T00:03:00Z,4
          """,
      },
    },
    #{ title: "No group-by dimensions" }
  )
  @opExample(
    #{
      returnType: #{
        contentType: "text/csv",
        _: """
          from,to,customer_id,customer_key,customer_name,model,type,value
          2023-01-01T00:00:00Z,2023-01-01T00:01:00Z,01G65Z755AFWAKHE12NY0CQ9FH,acme-inc,Acme Inc.,gpt-4-turbo,input,12
          2023-01-01T00:01:00Z,2023-01-01T00:02:00Z,01G65Z755AFWAKHE12NY0CQ9FH,acme-inc,Acme Inc.,gpt-4-turbo,input,20
          2023-01-01T00:02:00Z,2023-01-01T00:03:00Z,01G65Z755B3YZ1KRA9W8V7PS2D,globex,Globex Corp,gpt-4-turbo,output,4
          """,
      },
    },
    #{ title: "Grouped by customer and dimensions" }
  )
  @post
  @operationId("query-meter")
  @sharedRoute
  queryCsv(
    @path meterId: Shared.ULID,
    // Note: to avoid generating anyOf in the OpenAPI yaml, we omit the request body
    // @body request: MeterQueryRequest
  ): {
    @header contentType: "text/csv";

    @body
    _: string;
  } | Common.NotFound | Common.ErrorResponses;
}
