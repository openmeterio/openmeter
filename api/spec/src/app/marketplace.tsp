import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

import "../errors.tsp";
import "./oauth.tsp";
import "../types.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;

/**
 * Marketplace API.
 */
@route("/api/v1/marketplace")
@tag("Apps")
@friendlyName("Marketplace")
interface MarketplaceEndpoints {
  /**
   * List available apps of the app marketplace.
   */
  @get
  @route("/listings")
  @operationId("listMarketplaceListings")
  @summary("List available apps")
  list(
    ...QueryPagination,
  ): PaginatedResponse<MarketplaceListing> | CommonErrors;

  /**
   * Get a marketplace listing by type.
   */
  @get
  @route("/listings/{type}")
  @operationId("getMarketplaceListing")
  @summary("Get app details by type")
  get(type: AppType): MarketplaceListing | CommonErrors;

  /* **************** Install App via OAuth2 **************** */

  /**
   * Install an app via OAuth.
   * Returns a URL to start the OAuth 2.0 flow.
   */
  @get
  @route("/listings/{type}/install/oauth2")
  @operationId("marketplaceOAuth2InstallGetURL")
  @summary("Get OAuth2 install URL")
  getOAuth2InstallURL(
    @path type: AppType,
  ): OAuth2.ClientAppStartResponse | CommonErrors;

  /**
   * Authorize OAuth2 code.
   * Verifies the OAuth code and exchanges it for a token and refresh token
   */
  @get
  @route("/listings/{type}/install/oauth2/authorize")
  @operationId("marketplaceOAuth2InstallAuthorize")
  @summary("Install app via OAuth2")
  authorizeOAuth2Install(...MarketplaceOAuth2InstallAuthorizeRequest): {
    @statusCode _: 303;
  } | CommonErrors;

  /* **************** Install App via API Key **************** */

  /**
   * Install an marketplace app via API Key.
   */
  @post
  @route("/listings/{type}/install/apikey")
  @operationId("marketplaceAppAPIKeyInstall")
  @summary("Install app via API key")
  installWithAPIKey(
    ...MarketplaceApiKeyInstallRequest,
  ): MarketplaceInstallResponse | CommonErrors;

  /* **************** Install App **************** */

  /**
   * Install an app from the marketplace.
   */
  @post
  @route("/listings/{type}/install")
  @operationId("marketplaceAppInstall")
  @summary("Install app")
  install(
    ...MarketplaceInstallRequest,
  ): MarketplaceInstallResponse | CommonErrors;
}

/**
 * Marketplace install response.
 */
@friendlyName("MarketplaceInstallResponse")
model MarketplaceInstallResponse {
  app: App;

  /**
   * Default for capabilities
   */
  defaultForCapabilityTypes: AppCapabilityType[];
}

/**
 * Marketplace OAuth2 install request.
 */
#suppress "@openmeter/api-spec/casing" "Valid casing"
@friendlyName("MarketplaceOAuth2InstallAuthorizeRequest")
model MarketplaceOAuth2InstallAuthorizeRequest {
  ...OAuth2.AuthorizationCodeGrantParams;

  /**
   * The type of the app to install.
   */
  @path type: AppType;
}

/**
 * Marketplace install request payload.
 */
@friendlyName("MarketplaceInstallRequestPayload")
model MarketplaceInstallRequestPayload {
  /**
   * Name of the application to install.
   *
   * If name is not provided defaults to the marketplace listing's name.
   */
  name?: string;

  /**
   * If true, a billing profile will be created for the app.
   * The Stripe app will be also set as the default billing profile if the current default is a Sandbox app.
   */
  createBillingProfile?: boolean = true;
}

/**
 * App API key install request.
 */
@friendlyName("MarketplaceApiKeyInstallRequest")
model MarketplaceApiKeyInstallRequest {
  /**
   * The type of the app to install.
   */
  @path type: AppType;

  @body _: {
    ...MarketplaceInstallRequestPayload;

    /**
     * The API key for the provider.
     * For example, the Stripe API key.
     */
    apiKey: string;
  };
}

/**
 * Marketplace app install request.
 */
@friendlyName("MarketplaceInstallRequest")
model MarketplaceInstallRequest {
  /**
   * The type of the app to install.
   */
  @path type: AppType;

  @body _: MarketplaceInstallRequestPayload;
}

/**
 * A marketplace listing.
 * Represent an available app in the app marketplace that can be installed to the organization.
 *
 * Marketplace apps only exist in config so they don't extend the Resource model.
 */
@friendlyName("MarketplaceListing")
@example(#{
  type: AppType.Stripe,
  name: "Stripe",
  description: "Stripe integration allows you to collect payments with Stripe.",
  capabilities: #[
    #{
      type: AppCapabilityType.CalculateTax,
      key: "stripe_calculate_tax",
      name: "Calculate Tax",
      description: "Stripe Tax calculates tax portion of the invoices.",
    },
    #{
      type: AppCapabilityType.InvoiceCustomers,
      key: "stripe_invoice_customers",
      name: "Invoice Customers",
      description: "Stripe invoices customers with due amount.",
    },
    #{
      type: AppCapabilityType.CollectPayments,
      key: "stripe_collect_payments",
      name: "Collect Payments",
      description: "Stripe payments collects outstanding revenue with Stripe customer's default payment method.",
    }
  ],
  installMethods: #[AppInstallMethod.WithOAuth2, AppInstallMethod.WithAPIKey],
})
model MarketplaceListing {
  /**
   * The app's type
   */
  type: AppType;

  /**
   * The app's name.
   */
  name: string;

  /**
   * The app's description.
   */
  description: string;

  /**
   * The app's capabilities.
   */
  capabilities: AppCapability[];

  /**
   * Install methods.
   *
   * List of methods to install the app.
   */
  installMethods: AppInstallMethod[];
}

/**
 * Install method of the application.
 */
@friendlyName("InstallMethod")
enum AppInstallMethod {
  WithOAuth2: "with_oauth2",
  WithAPIKey: "with_api_key",
  NoCredentialsRequired: "no_credentials_required",
}
