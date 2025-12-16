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
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  create(
    @body meter: Shared.CreateRequest<Meter>,
  ): Shared.CreateResponse<Meter> | Common.ErrorResponses;

  /**
   * Get a meter by ID.
   */
  @get
  @operationId("get-meter")
  @summary("Get meter")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  get(
    @path meterId: Shared.ULID,
  ): Shared.GetResponse<Meter> | Common.NotFound | Common.ErrorResponses;

  /**
   * List meters.
   */
  @get
  @operationId("list-meters")
  @summary("List meters")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  list(
    ...Common.PagePaginationQuery,
  ): Shared.PagePaginatedResponse<Meter> | Common.ErrorResponses;

  /**
   * Delete a meter.
   */
  @delete
  @operationId("delete-meter")
  @summary("Delete meter")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  delete(
    @path meterId: Shared.ULID,
  ): Shared.DeleteResponse | Common.NotFound | Common.ErrorResponses;
}
