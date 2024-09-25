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
  type: AppCapabilityType.CollectPayments,
  description: "Stripe payments collects outstanding revenue with Stripe customer's default payment method.",
})
model AppCapability {
  /**
   * The capability type.
   */
  type: AppCapabilityType;

  /**
   * The capability description.
   */
  description: string;
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
  Ready: "ready",

  // TODO
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
  capabilities: #[AppCapabilityType.CollectPayments],
  requirements: #[AppRequirements.CustomerCountryCode],
})
model AppConnected {
  ...Resource;

  /**
   * The app's type
   */
  type: AppType;

  /**
   * Status of the app connection.
   */
  status: AppConnectedStatus;

  /**
   * The app's installed capabilities.
   * This is a subset of the app gallery's capabilities.
   */
  capabilities: AppCapabilityType[];

  /**
   * Requirements
   */
  requirements: AppRequirements[];
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
    AppCapabilityType.CalculateTax,
    AppCapabilityType.InvoiceCustomers,
    AppCapabilityType.CollectPayments,
    AppCapabilityType.ReportUsage
  ],
  requirements: #[AppRequirements.CustomerCountryCode, AppRequirements.CustomerExternalStripeCustomerId],

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
