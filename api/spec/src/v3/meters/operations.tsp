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
  @post
  @operationId("create-meter")
  @summary("Create meter")
  create(
    @body meter: Shared.CreateRequest<Meter>,
  ): Shared.CreateResponse<Meter> | Common.ErrorResponses;

  @get
  @operationId("get-meter")
  @summary("Get meter")
  get(
    @path meterId: Shared.ULID,
  ): Shared.GetResponse<Meter> | Common.NotFound | Common.ErrorResponses;

  @get
  @operationId("list-meters")
  @summary("List meters")
  list(
    ...Common.CursorPageQuery,
  ): Shared.CursorPaginatedResponse<Meter> | Common.ErrorResponses;
}
