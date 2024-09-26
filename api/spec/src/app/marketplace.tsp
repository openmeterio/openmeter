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
  @route("/listings")
  @operationId("marketplaceListListings")
  marketplaceListings(...MarketplaceListListingsRequest): Paginated<Listing>[] | CommonErrors;

  /* **************** Install App via OAuth2 **************** */

  /**
   * Install an app via OAuth.
   * Returns a URL to start the OAuth 2.0 flow.
   */
  @get
  @route("/install/oauth2")
  @operationId("marketplaceOAuth2InstallGetURL")
  marketplaceOAuth2InstallGetURL(@query appKey: Key): OAuth2.ClientAppStartResponse | CommonErrors;

  /**
   * Authorize OAuth2 code.
   * Verifies the OAuth code and exchanges it for a token and refresh token
   */
  @get
  @route("/install/oauth2/authorize")
  @operationId("marketplaceOAuth2InstallAuthorize")
  marketplaceOAuth2InstallAuthorize(
    ...MarketplaceOAuth2InstallAuthorizeRequest,
  ): MarketplaceOAuth2InstallAuthorizeResponse | CommonErrors;

  /* **************** Install App via API Key **************** */

  /**
   * Install an marketplace via API Key.
   */
  @post
  @route("/install/apikey")
  @operationId("marketplaceAPIKeyInstall")
  appAPIKeyInstall(...MarketplaceApiKeyInstallRequest): App | CommonErrors;
}

/**
 * Query params for listing marketplace listings.
 */
@friendlyName("MarketplaceListListingsRequest")
model MarketplaceListListingsRequest extends PaginatedQuery {}

/**
 * Marketplace OAuth2 install request.
 */
@friendlyName("MarketplaceOAuth2InstallAuthorizeRequest")
model MarketplaceOAuth2InstallAuthorizeRequest {
  ...OAuth2.AuthorizationCodeGrantParams;

  /**
   * The key of the app to install.
   */
  @query appKey: Key;
}

/**
 * Marketplace OAuth2 install response.
 */
@friendlyName("MarketplaceOAuth2InstallAuthorizeResponse")
model MarketplaceOAuth2InstallAuthorizeResponse {
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
 * A marketplace listing.
 * Represent an available app in the app marketplace that can be installed to the organization.
 *
 * Marketplace apps only exist in config so they don't extend the Resource model.
 */
@friendlyName("MarketplaceListing")
@example(#{
  key: "stripe",
  type: AppType.Stripe,
  name: "Stripe",
  description: "Stripe interation allows you to collect payments with Stripe.",
  iconUrl: "/images/stripe.png",
  capabilities: #[
    #{
      type: CapabilityType.CalculateTax,
      key: "stripe_calculate_tax",
      name: "Calculate Tax",
      description: "Stripe Tax calculates tax portion of the invoices.",
      requirements: #[Requirements.CustomerExternalStripeCustomerId],
    },
    #{
      type: CapabilityType.InvoiceCustomers,
      key: "stripe_invoice_customers",
      name: "Invoice Customers",
      description: "Stripe invoices customers with due amount.",
      requirements: #[Requirements.CustomerExternalStripeCustomerId],
    },
    #{
      type: CapabilityType.CollectPayments,
      key: "stripe_collect_payments",
      name: "Collect Payments",
      description: "Stripe payments collects outstanding revenue with Stripe customer's default payment method.",
      requirements: #[Requirements.CustomerExternalStripeCustomerId],
    }
  ],
})
model Listing {
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
  capabilities: Capability[];
}
