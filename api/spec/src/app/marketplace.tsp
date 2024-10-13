import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

import "../errors.tsp";
import "./oauth.tsp";
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
interface Marketplace {
  /**
   * List available apps of the app marketplace.
   */
  @get
  @route("/listings")
  @operationId("marketplaceListListings")
  list(...PaginatedQuery): Paginated<MarketplaceListing> | CommonErrors;

  /**
   * Get a marketplace listing by type.
   */
  @get
  @route("/listings/{type}")
  @operationId("getMarketplaceListing")
  get(type: Type): MarketplaceListing | CommonErrors;

  /* **************** Install App via OAuth2 **************** */

  /**
   * Install an app via OAuth.
   * Returns a URL to start the OAuth 2.0 flow.
   */
  @get
  @route("/listings/{type}/install/oauth2")
  @operationId("marketplaceOAuth2InstallGetURL")
  getOAuth2InstallURL(@path type: Type): OAuth2.ClientAppStartResponse | CommonErrors;

  /**
   * Authorize OAuth2 code.
   * Verifies the OAuth code and exchanges it for a token and refresh token
   */
  @get
  @route("/listings/{type}/install/oauth2/authorize")
  @operationId("marketplaceOAuth2InstallAuthorize")
  authorizeOAuth2Install(...MarketplaceOAuth2InstallAuthorizeRequest): {
    @statusCode _: 303;
  } | CommonErrors;

  /* **************** Install App via API Key **************** */

  /**
   * Install an marketplace via API Key.
   */
  @post
  @route("/listings/{type}/install/apikey")
  @operationId("marketplaceAppAPIKeyInstall")
  installWithAPIKey(...MarketplaceApiKeyInstallRequest): AppBase | CommonErrors;
}

/**
 * Marketplace OAuth2 install request.
 */
@friendlyName("MarketplaceOAuth2InstallAuthorizeRequest")
model MarketplaceOAuth2InstallAuthorizeRequest {
  ...OAuth2.AuthorizationCodeGrantParams;

  /**
   * The type of the app to install.
   */
  @path type: Type;
}

/**
 * App API key install request.
 */
@friendlyName("MarketplaceApiKeyInstallRequest")
model MarketplaceApiKeyInstallRequest {
  /**
   * The type of the app to install.
   */
  @path type: Type;

  @body _: {
    /**
     * The API key for the provider.
     * For example, the Stripe API key.
     */
    apiKey: string;
  };
}

/**
 * A marketplace listing.
 * Represent an available app in the app marketplace that can be installed to the organization.
 *
 * Marketplace apps only exist in config so they don't extend the Resource model.
 */
@friendlyName("MarketplaceListing")
@example(#{
  type: Type.Stripe,
  name: "Stripe",
  description: "Stripe interation allows you to collect payments with Stripe.",
  iconUrl: "/images/stripe.png",
  capabilities: #[
    #{
      type: CapabilityType.CalculateTax,
      key: "stripe_calculate_tax",
      name: "Calculate Tax",
      description: "Stripe Tax calculates tax portion of the invoices.",
    },
    #{
      type: CapabilityType.InvoiceCustomers,
      key: "stripe_invoice_customers",
      name: "Invoice Customers",
      description: "Stripe invoices customers with due amount.",
    },
    #{
      type: CapabilityType.CollectPayments,
      key: "stripe_collect_payments",
      name: "Collect Payments",
      description: "Stripe payments collects outstanding revenue with Stripe customer's default payment method.",
    }
  ],
})
model MarketplaceListing {
  /**
   * The app's type
   */
  type: Type;

  /**
   * The app's name.
   */
  name: string;

  /**
   * The app's description.
   */
  description: string;

  /**
   * The app's icon URL.
   */
  iconUrl: string;

  /**
   * The app's capabilities.
   */
  capabilities: Capability[];
}
