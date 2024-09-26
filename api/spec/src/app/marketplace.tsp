import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

import "../errors.tsp";
import "../oauth.tsp";
import "../pagination.tsp";
import "../types.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.App;

/**
 * Marketplace API.
 */
@route("/api/v1/marketplace")
@tag("Apps")
interface MarketplaceAPI {
  /**
   * List available apps of the app marketplace.
   */
  @get
  @route("/apps")
  @operationId("marketplaceListApps")
  marketplaceList(...MarketplaceListAppsListRequest): Paginated<MarketplaceApp>[] | CommonErrors;

  /* **************** Install App via OAuth2 **************** */

  /**
   * Install an app via OAuth.
   * Returns a URL to start the OAuth 2.0 flow.
   */
  @get
  @route("/install/oauth2")
  @operationId("appOAuth2Install")
  oAuth2Install(@query appKey: Key): OAuth2.ClientAppStartResponse | CommonErrors;

  /**
   * Authorize OAuth2 code.
   * Verifies the OAuth code and exchanges it for a token and refresh token
   */
  @get
  @route("/install/oauth2/authorize")
  @operationId("appOAuth2Authorize")
  oAuth2Authorize(...MarketplaceOAuth2InstallRequest): MarketplaceOAuth2InstallResponse | CommonErrors;

  /* **************** Install App via API Key **************** */

  /**
   * Install an app via API Key.
   */
  @post
  @route("/install/apikey")
  @operationId("appApiKeyInstall")
  apiKeyInstall(...MarketplaceApiKeyInstallRequest): ConnectedApp | CommonErrors;
}

/**
 * Query params for listing app marketplace.
 */
@friendlyName("MarketplaceListAppsListRequest")
model MarketplaceListAppsListRequest extends PaginatedQuery {}

/**
 * Marketplace OAuth2 install request.
 */
@friendlyName("MarketplaceOAuth2InstallRequest")
model MarketplaceOAuth2InstallRequest {
  ...OAuth2.AuthorizationCodeGrantParams;

  /**
   * The key of the app to install.
   */
  @query appKey: Key;
}

/**
 * Marketplace OAuth2 install response.
 */
@friendlyName("MarketplaceOauth2AuthorizationCodeGrantResponse")
model MarketplaceOAuth2InstallResponse {
  @statusCode _: 303;
}

/**
 * App API key install request.
 */
@friendlyName("MarketplaceApiKeyInstallRequest")
model MarketplaceApiKeyInstallRequest {
  @body _: {
    /**
     * The key of the app to install.
     */
    appKey: Key;

    /**
     * The API key.
     */
    apiKey: string;
  };
}

/**
 * A marketplace app object.
 * Represent an available app in the app marketplace that can be installed to the organization.
 *
 * Marketplace apps only exist in config so they don't extend the Resource model.
 */
@friendlyName("MarketplaceApp")
@example(#{
  key: "stripe",
  type: AppType.Stripe,
  name: "Stripe",
  description: "Stripe interation allows you to collect payments with Stripe.",
  iconUrl: "/images/stripe.png",
  capabilities: #[
    #{
      type: AppCapabilityType.CollectPayments,
      key: "collect_payment",
      name: "Collect Payments",
      description: "Stripe payments collects outstanding revenue with Stripe customer's default payment method.",
    }
  ],
})
model MarketplaceApp {
  /**
   * The app's key.
   */
  key: Key;

  /**
   * The app's name.
   */
  name: string;

  /**
   * The app's description.
   */
  description: string;

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
