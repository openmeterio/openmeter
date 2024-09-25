import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

import "./errors.tsp";
import "./oauth.tsp";
import "./pagination.tsp";
import "./types.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;

/**
 * App Gallery API.
 */
@route("/api/v1/apps/gallery")
@tag("Apps")
interface AppGalleryAPI {
  /**
   * List available apps of the app gallery.
   */
  @get
  @operationId("appGalleryList")
  galleryList(...AppGalleryListParams): Paginated<AppGalleryItem>[] | CommonErrors;

  /* **************** Install via OAuth2 **************** */

  /**
   * Install an app via OAuth.
   * Returns a URL to start the OAuth 2.0 flow.
   */
  @get
  @route("/{id}/oauth2/install")
  @operationId("appOAuth2Install")
  oAuth2Install(@path id: string): OAuth2.ClientAppStartResponse | CommonErrors;

  /**
   * Authorize OAuth2 code.
   * Verifies the OAuth code and exchanges it for a token and refresh token
   */
  @get
  @route("/{id}/oauth2/authorize")
  @operationId("appOAuth2Authorize")
  oAuth2Authorize(
    @path id: string,
    ...OAuth2.AuthorizationCodeGrantParams,
  ): AppOauth2AuthorizationCodeGrantResponse | CommonErrors;

  /* **************** Install via API Key **************** */

  /**
   * Install an app via API Key.
   */
  @post
  @route("/{id}/apikey/install")
  @operationId("appApiKeyInstall")
  apiKeyInstall(@path id: string, ...AppApiKeyInstallRequest): AppConnected | CommonErrors;
}

/**
 * App OAuth2 authorization code grant response.
 */
@friendlyName("AppOauth2AuthorizationCodeGrantResponse")
model AppOauth2AuthorizationCodeGrantResponse {
  @statusCode _: 303;
}

/**
 * App API key install request.
 */
model AppApiKeyInstallRequest {
  /**
   * The API key.
   */
  @body
  apiKey: string;
}

/**
 * Query params for listing app gallery.
 */
@friendlyName("AppGalleryListParams")
model AppGalleryListParams extends PaginatedQuery {}

/**
 * App Gallery API.
 */
@route("/api/v1/apps/connected")
@tag("Apps")
interface AppConnectedAPI {
  /**
   * Get a connected app.
   */
  @get
  @route("/connected/{id}")
  @operationId("appConnectedGet")
  connectedGet(@path id: string): AppConnected | CommonErrors;

  /**
   * Uninstall a connected app.
   */
  @delete
  @route("/connected/{id}")
  @operationId("appConnectedUninstall")
  connectedUninstall(@path id: string): AppConnected | NotFoundError | CommonErrors;
}

/**
 * Query params for listing app gallery.
 */
@friendlyName("AppConnectedListParams")
model AppConnectedListParams extends PaginatedQuery {}

/**
 * App type.
 */
enum AppType {
  Stripe: "stripe",
}

/**
 * App capability.
 */
@friendlyName("AppCapability")
@example(#{
  id: "01G65Z755AFWAKHE12NY0CQ9FH",
  name: "Collect Payments",
  description: "Stripe payments collects outstanding revenue with Stripe customer's default payment method.",
  createdAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
  updatedAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
  type: AppCapabilityType.CollectPayments,
})
model AppCapability {
  ...Resource;

  /**
   * The capability type.
   */
  type: AppCapabilityType;

  /**
   * The capability requirements.
   */
  requirements?: AppRequirements[];
}

/**
 * App capability type.
 */
@friendlyName("AppCapabilityType")
enum AppCapabilityType {
  /**
   * The app can report aggregated usage.
   */
  ReportUsage: "reportUsage",

  /**
   * The app can report raw events.
   */
  ReportEvents: "reportEvents",

  /**
   * The app can calculate tax.
   */
  CalculateTax: "calculateTax",

  /**
   * The app can invoice customers.
   */
  InvoiceCustomers: "invoiceCustomers",

  /**
   * The app can collect payments.
   */
  CollectPayments: "collectPayments",
}

/**
 * App requirements.
 */
@friendlyName("AppRequirements")
enum AppRequirements {
  CustomerCountryCode: "customer.countryCode",
  CustomerExternalStripeCustomerId: "customer.external.stripeCustomerId",
}

/**
 * An app gallery item.
 * Represent an available app in the app gallery, ready to be connected to the organization.
 */
@friendlyName("App")
@example(#{
  id: "01G65Z755AFWAKHE12NY0CQ9FH",
  type: AppType.Stripe,
  name: "Stripe",
  createdAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
  updatedAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
  description: "Stripe interation allows you to collect payments with Stripe.",
  iconUrl: "/images/stripe.png",
  capabilities: #[
    #{
      id: "01G65Z755AFWAKHE12NY0CQ9FH",
      name: "Collect Payments",
      createdAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
      updatedAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
      type: AppCapabilityType.CollectPayments,
      description: "Stripe payments collects outstanding revenue with Stripe customer's default payment method.",
    }
  ],
})
model AppGalleryItem {
  ...Resource;

  /**
   * The app's type
   */
  type: AppType;

  /**
   * The app's icon URL.
   */
  iconUrl: string;

  /**
   * The app's capabilities.
   */
  capabilities: AppCapability[];
}

/**
 * App connected status.
 */
enum AppConnectedStatus {
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
 * A connected app object.
 * Represent an app connected to the organization.
 * This is an actual instance, with its own configuration and credentials.
 */
@friendlyName("AppConnected")
@example(#{
  id: "01G65Z755AFWAKHE12NY0CQ9FH",
  type: AppType.Stripe,
  name: "Stripe",
  createdAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
  updatedAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
  status: AppConnectedStatus.Ready,
  capabilities: #[
    #{
      id: "01G65Z755AFWAKHE12NY0CQ9FH",
      name: "Collect Payments",
      createdAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
      updatedAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
      type: AppCapabilityType.CollectPayments,
      description: "Stripe payments collects outstanding revenue with Stripe customer's default payment method.",
      requirements: #[AppRequirements.CustomerExternalStripeCustomerId],
    }
  ],
})
model AppConnected {
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
  status: AppConnectedStatus;

  /**
   * The app's installed capabilities.
   * This is a subset of the app gallery's capabilities.
   */
  @visibility("read")
  capabilities: AppCapability[];
}

/**
 * A connected Stripe app object.
 */
@friendlyName("AppStripe")
@example(#{
  id: "01G65Z755AFWAKHE12NY0CQ9FH",
  type: AppType.Stripe,
  name: "Stripe",
  createdAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
  updatedAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
  status: AppConnectedStatus.Ready,
  capabilities: #[
    #{
      id: "01G65Z755AFWAKHE12NY0CQ9FH",
      name: "Calculate Tax",
      createdAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
      updatedAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
      type: AppCapabilityType.CollectPayments,
      description: "Stripe Tax calculates tax portion of the invoices.",
      requirements: #[AppRequirements.CustomerExternalStripeCustomerId],
    },
    #{
      id: "01G65Z755AFWAKHE12NY0CQ9FH",
      name: "Invoice Customers",
      createdAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
      updatedAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
      type: AppCapabilityType.CollectPayments,
      description: "Stripe invoices customers with due amount.",
      requirements: #[AppRequirements.CustomerExternalStripeCustomerId],
    },
    #{
      id: "01G65Z755AFWAKHE12NY0CQ9FH",
      name: "Collect Payments",
      createdAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
      updatedAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
      type: AppCapabilityType.CollectPayments,
      description: "Stripe payments collects outstanding revenue with Stripe customer's default payment method.",
      requirements: #[AppRequirements.CustomerExternalStripeCustomerId],
    }
  ],

  // Stripe specific fields
  accountId: "acct_123456789",

  livemode: true,
})
model AppConnectedStripe {
  ...AppConnected;

  /**
   * The Stripe account ID.
   */
  accountId: string;

  /**
   * Livemode, true if the app is in production mode.
   */
  livemode: boolean;
}
