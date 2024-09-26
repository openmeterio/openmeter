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
 * ConnectedApp.
 * One of: stripe
 */
@friendlyName("ConnectedApp")
@discriminator("type")
union ConnectedApp {
  @summary("Stripe")
  stripe: StripeConnectedApp,
}

/**
 * Abstract base model for connected apps.
 *
 * Represent an app connected to the organization.
 * This is an actual instance, with its own configuration and credentials.
 */
@friendlyName("ConnectedAppBase")
model ConnectedAppBase {
  ...Resource;

  /**
   * The marketplace app that this connected app is based on.
   */
  @visibility("read")
  marketplaceApp: MarketplaceApp;

  /**
   * Status of the app connection.
   */
  @visibility("read")
  status: ConnectedAppStatus;
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

/**
 * A connected Stripe app object.
 */
@friendlyName("StripeConnectedApp")
@example(#{
  id: "01G65Z755AFWAKHE12NY0CQ9FH",
  type: AppType.Stripe,
  name: "Stripe",
  status: ConnectedAppStatus.Ready,
  marketplaceApp: #{
    key: "stripe",
    type: AppType.Stripe,
    name: "Stripe",
    description: "Stripe interation allows you to collect payments with Stripe.",
    iconUrl: "/images/stripe.png",
    capabilities: #[
      #{
        type: AppCapabilityType.CalculateTax,
        key: "stripe_calculate_tax",
        name: "Calculate Tax",
        description: "Stripe Tax calculates tax portion of the invoices.",
        requirements: #[AppRequirements.CustomerExternalStripeCustomerId],
      },
      #{
        type: AppCapabilityType.InvoiceCustomers,
        key: "stripe_invoice_customers",
        name: "Invoice Customers",
        description: "Stripe invoices customers with due amount.",
        requirements: #[AppRequirements.CustomerExternalStripeCustomerId],
      },
      #{
        type: AppCapabilityType.CollectPayments,
        key: "stripe_collect_payments",
        name: "Collect Payments",
        description: "Stripe payments collects outstanding revenue with Stripe customer's default payment method.",
        requirements: #[AppRequirements.CustomerExternalStripeCustomerId],
      }
    ],
  },
  createdAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
  updatedAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),

  // Stripe specific fields
  accountId: "acct_123456789",

  livemode: true,
})
model StripeConnectedApp {
  ...ConnectedAppBase;

  /**
   * The app's type is Stripe.
   */
  type: AppType.Stripe;

  /**
   * The Stripe account ID.
   */
  accountId: string;

  /**
   * Livemode, true if the app is in production mode.
   */
  livemode: boolean;
}
