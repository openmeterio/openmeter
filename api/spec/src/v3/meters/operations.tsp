import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./meter.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Meters;

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
   * Get a meter by ID or key.
   */
  @get
  @operationId("get-meter")
  @summary("Get meter")
  get(
    @path meterIdOrKey: Shared.ULIDOrResourceKey,
  ): Shared.GetResponse<Meter> | Common.NotFound | Common.ErrorResponses;

  /**
   * List meters.
   */
  @get
  @operationId("list-meters")
  @summary("List meters")
  list(
    ...Common.CursorPageQuery,
  ): Shared.CursorPaginatedResponse<Meter> | Common.ErrorResponses;
}
