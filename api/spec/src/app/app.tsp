import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";

import "./app.tsp";
import "./capability.tsp";
import "../errors.tsp";
import "../oauth.tsp";
import "../pagination.tsp";
import "./stripe.tsp";
import "../types.tsp";

using OpenMeter.App;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.App;

/**
 * Installed Apps API.
 */
@route("/api/v1/apps")
@tag("Apps")
interface Apps {
  /**
   * List apps.
   */
  @get
  @route("/")
  @operationId("listApps")
  list(...ListAppsRequest): Paginated<App>[] | CommonErrors;

  /**
   * Get the app.
   */
  @get
  @route("/{id}")
  @operationId("getApp")
  get(@path id: ULID): App | NotFoundError | CommonErrors;

  /**
   * Uninstall an app.
   */
  @delete
  @route("/{id}")
  @operationId("uninstallApp")
  uninstall(@path id: ULID): {
    @statusCode _: 204;
  } | NotFoundError | CommonErrors;
}

/**
 * Query params for listing installed apps
 */
@friendlyName("ListAppsRequest")
model ListAppsRequest extends PaginatedQuery {}

/**
 * Type of the app.
 */
enum Type {
  Stripe: "stripe",
}

/**
 * App.
 * One of: stripe
 */
@friendlyName("App")
@discriminator("type")
union App {
  stripe: StripeApp,
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
  @visibility("read")
  listing: Listing;

  /**
   * Status of the app connection.
   */
  @visibility("read")
  status: AppStatus;
}

/**
 * App installed status.
 */
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
