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

interface AppsOperations {
  /**
   * List installed apps.
   */
  @get
  @operationId("list-apps")
  @summary("List apps")
  @extension(Shared.InternalExtension, true)
  @extension(Shared.UnstableExtension, true)
  list(...Common.PagePaginationQuery): Shared.PagePaginatedResponse<App> | Common.ErrorResponses;

  /**
   * Get an installed app.
   */
  @get
  @operationId("get-app")
  @summary("Get app")
  @extension(Shared.InternalExtension, true)
  @extension(Shared.UnstableExtension, true)
  get(@path appId: Shared.ULID): Shared.GetResponse<App> | Common.NotFound | Common.ErrorResponses;
}
