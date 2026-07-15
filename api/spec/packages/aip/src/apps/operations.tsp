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
  list(...Common.PagePaginationQuery):
    | Shared.PagePaginatedResponse<App>
    | Common.ErrorResponses;

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

interface AppCatalogOperations {
  /**
   * List available apps.
   */
  @get
  @operationId("list-app-catalog")
  @summary("List app catalog")
  list(...Common.PagePaginationQuery):
    | Shared.PagePaginatedResponse<AppCatalogItem>
    | Common.ErrorResponses;

  /**
   * Get an app catalog item by type.
   */
  @get
  @route("/{appType}")
  @operationId("get-app-catalog-item")
  @summary("Get app catalog item by type")
  get(@path appType: AppType):
    | AppCatalogItem
    | Common.NotFound
    | Common.ErrorResponses;

  /**
   * Install an app from the catalog.
   */
  @post
  @route("/install")
  @operationId("install-app")
  @summary("Install app from the catalog")
  install(@body body: InstallAppRequest):
    | Shared.GetResponse<InstallAppResponse>
    | Common.ErrorResponses;
}
