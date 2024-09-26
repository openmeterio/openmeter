import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

import "./app.tsp";
import "./capability.tsp";
import "../errors.tsp";
import "../oauth.tsp";
import "../pagination.tsp";
import "../types.tsp";

using OpenMeter.App;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.App;

/**
 * Connected Apps API.
 */
@route("/api/v1/connected-apps")
@tag("Apps")
interface ConnectedAppAPI {
  /**
   * List connected apps.
   */
  @get
  @route("/")
  @operationId("connectedAppList")
  connectedList(...ConnectedAppListRequest): Paginated<ConnectedApp>[] | CommonErrors;

  /**
   * Get a connected app.
   */
  @get
  @route("/{id}")
  @operationId("connectedAppGet")
  connectedGet(@path id: string): ConnectedApp | CommonErrors;

  /**
   * Uninstall a connected app.
   */
  @delete
  @route("/{id}")
  @operationId("connectedAppUninstall")
  connectedUninstall(@path id: string): ConnectedApp | NotFoundError | CommonErrors;
}

/**
 * Query params for listing connected apps
 */
@friendlyName("ConnectedAppListRequest")
model ConnectedAppListRequest extends PaginatedQuery {}

/**
 * A connected app object.
 * Represent an app connected to the organization.
 * This is an actual instance, with its own configuration and credentials.
 */
@friendlyName("ConnectedApp")
@example(#{
  id: "01G65Z755AFWAKHE12NY0CQ9FH",
  type: AppType.Stripe,
  name: "Stripe",
  createdAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
  updatedAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
  status: ConnectedAppStatus.Ready,
  capabilities: #[
    #{
      type: AppCapabilityType.CollectPayments,
      key: "stripe_collect_payments",
      name: "Collect Payments",
      description: "Stripe payments collects outstanding revenue with Stripe customer's default payment method.",
      requirements: #[AppRequirements.CustomerExternalStripeCustomerId],
    }
  ],
})
model ConnectedApp {
  ...Resource;

  /**
   * The app's type
   */
  @visibility("read")
  type: AppType;

  /**
   * Status of the app connection.
   */
  @visibility("read")
  status: ConnectedAppStatus;

  /**
   * The app's installed capabilities.
   * This is a subset of the app gallery's capabilities.
   */
  @visibility("read")
  capabilities: AppCapability[];
}

/**
 * App connected status.
 */
enum ConnectedAppStatus {
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
