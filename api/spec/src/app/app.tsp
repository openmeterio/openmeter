import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";

import "./app.tsp";
import "./capability.tsp";
import "../errors.tsp";
import "./stripe.tsp";
import "../types.tsp";
import "../query.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;

/**
 * Installed Apps API.
 */
@route("/api/v1/apps")
@tag("Apps")
@friendlyName("Apps")
interface AppsEndpoints {
  /**
   * List apps.
   */
  @get
  @operationId("listApps")
  @summary("List apps")
  list(...ListAppsRequest): PaginatedResponse<App> | CommonErrors;

  /**
   * Get the app.
   */
  @get
  @route("/{id}")
  @operationId("getApp")
  @summary("Get app")
  get(@path id: ULID): App | NotFoundError | CommonErrors;

  /**
   * Update an app.
   */
  @put
  @route("/{id}")
  @operationId("updateApp")
  @summary("Update app")
  update(
    @path id: ULID,
    @body app: AppReplaceUpdate,
  ): App | NotFoundError | CommonErrors;

  /**
   * Uninstall an app.
   */
  @delete
  @route("/{id}")
  @operationId("uninstallApp")
  @summary("Uninstall app")
  uninstall(@path id: ULID): {
    @statusCode _: 204;
  } | NotFoundError | CommonErrors;
}

/**
 * Query params for listing installed apps
 */
@friendlyName("ListAppsRequest")
model ListAppsRequest {
  ...QueryPagination;
}

/**
 * Type of the app.
 */
@friendlyName("AppType")
enum AppType {
  Stripe: "stripe",
  Sandbox: "sandbox",
  CustomInvoicing: "custom_invoicing",
}

/**
 * App.
 * One of: stripe
 */
@friendlyName("App")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union App {
  stripe: StripeApp,
  sandbox: SandboxApp,
  custom_invoicing: CustomInvoicingApp,
}

/**
 * App ReplaceUpdate Model
 */
// TODO: Once we can properly do ReplaceModel<App> we can use that here (right now we cannot do this recursively and it fails on Resource).
@friendlyName("AppReplaceUpdate")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union AppReplaceUpdate {
  stripe: TypeSpec.Rest.Resource.ResourceReplaceModel<StripeApp>,
  sandbox: TypeSpec.Rest.Resource.ResourceReplaceModel<SandboxApp>,
  custom_invoicing: TypeSpec.Rest.Resource.ResourceReplaceModel<CustomInvoicingApp>,
}

/**
 * App reference
 *
 * Can be used as a short reference to an app if the full app object is not needed.
 */
@friendlyName("AppReference")
model AppReference {
  /**
   * The ID of the app.
   */
  id: ULID;
}

/**
 * Abstract base model for installed apps.
 *
 * Represent an app installed to the organization.
 * This is an actual instance, with its own configuration and credentials.
 */
@friendlyName("AppBase")
model AppBase {
  ...Resource;

  /**
   * The marketplace listing that this installed app is based on.
   */
  @visibility(Lifecycle.Read)
  listing: MarketplaceListing;

  /**
   * Status of the app connection.
   */
  @visibility(Lifecycle.Read)
  status: AppStatus;
}

/**
 * App installed status.
 */
@friendlyName("AppStatus")
enum AppStatus {
  /**
   * The app is ready to be used.
   */
  Ready: "ready",

  /**
   * The app is unauthorized.
   * This usually happens when the app's credentials are revoked or expired.
   * To resolve this, the user must re-authorize the app.
   */
  Unauthorized: "unauthorized",
}
