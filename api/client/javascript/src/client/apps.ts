import { transformResponse } from './utils.js'
import type { RequestOptions } from './common.js'
import type {
  AppReplaceUpdate,
  CreateStripeCheckoutSessionRequest,
  operations,
  paths,
} from './schemas.js'
import type { Client } from 'openapi-fetch'

/**
 * Apps
 * Manage integrations for extending OpenMeter's functionality.
 */
export class Apps {
  public marketplace: AppMarketplace
  public stripe: AppStripe

  constructor(private client: Client<paths, `${string}/${string}`>) {
    this.marketplace = new AppMarketplace(client)
    this.stripe = new AppStripe(client)
  }

  /**
   * List apps
   * @param query - The query parameters
   * @param signal - An optional abort signal
   * @returns The apps
   */
  public async list(
    query?: operations['listApps']['parameters']['query'],
    options?: RequestOptions
  ) {
    const resp = await this.client.GET('/api/v1/apps', {
      params: { query },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get an app
   * @param id - The ID of the app
   * @param signal - An optional abort signal
   * @returns The app
   */
  public async get(
    id: operations['getApp']['parameters']['path']['id'],
    options?: RequestOptions
  ) {
    const resp = await this.client.GET('/api/v1/apps/{id}', {
      params: { path: { id } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Update an app
   * @param id - The ID of the app
   * @param body - The body of the request
   * @param signal - An optional abort signal
   * @returns The app
   */
  public async update(
    id: operations['updateApp']['parameters']['path']['id'],
    body: AppReplaceUpdate,
    options?: RequestOptions
  ) {
    const resp = await this.client.PUT('/api/v1/apps/{id}', {
      body,
      params: { path: { id } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Uninstall an app
   * @param id - The ID of the app
   * @param signal - An optional abort signal
   * @returns The app
   */
  public async uninstall(
    id: operations['uninstallApp']['parameters']['path']['id'],
    options?: RequestOptions
  ) {
    const resp = await this.client.DELETE('/api/v1/apps/{id}', {
      params: { path: { id } },
      ...options,
    })

    return transformResponse(resp)
  }
}

/**
 * App Marketplace
 * Available apps from the OpenMeter Marketplace.
 */
export class AppMarketplace {
  constructor(private client: Client<paths, `${string}/${string}`>) { }

  /**
   * List available apps
   * @param query - The query parameters
   * @param signal - An optional abort signal
   * @returns The apps
   */
  public async list(
    query?: operations['listMarketplaceListings']['parameters']['query'],
    options?: RequestOptions
  ) {
    const resp = await this.client.GET('/api/v1/marketplace/listings', {
      params: { query },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get details for a listing
   * @param type - The type of the listing
   * @param signal - An optional abort signal
   * @returns The listing
   */
  public async get(
    type: operations['getMarketplaceListing']['parameters']['path']['type'],
    options?: RequestOptions
  ) {
    const resp = await this.client.GET('/api/v1/marketplace/listings/{type}', {
      params: { path: { type } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Install an app via OAuth. Returns a URL to start the OAuth 2.0 flow.
   * @param type - The type of the listing
   * @param signal - An optional abort signal
   * @returns The OAuth2 install URL
   */
  public async getOauth2InstallUrl(
    type: operations['marketplaceOAuth2InstallGetURL']['parameters']['path']['type'],
    options?: RequestOptions
  ) {
    const resp = await this.client.GET(
      '/api/v1/marketplace/listings/{type}/install/oauth2',
      {
        params: { path: { type } },
        ...options,
      }
    )

    return transformResponse(resp)
  }

  /**
   * Authorize OAuth2 code. Verifies the OAuth code and exchanges it for a token and refresh token
   * @param type - The type of the listing
   * @param signal - An optional abort signal
   * @returns The authorization URL
   */
  public async authorizeOauth2(
    type: operations['marketplaceOAuth2InstallAuthorize']['parameters']['path']['type'],
    options?: RequestOptions
  ) {
    const resp = await this.client.GET(
      '/api/v1/marketplace/listings/{type}/install/oauth2/authorize',
      {
        params: { path: { type } },
        ...options,
      }
    )

    return transformResponse(resp)
  }

  /**
   * Install an app via API key.
   * @param type - The type of the listing
   * @param signal - An optional abort signal
   * @returns The installation
   */
  public async installWithAPIKey(
    type: operations['marketplaceAppAPIKeyInstall']['parameters']['path']['type'],
    body: operations['marketplaceAppAPIKeyInstall']['requestBody']['content']['application/json'],
    options?: RequestOptions
  ) {
    const resp = await this.client.POST(
      '/api/v1/marketplace/listings/{type}/install/apikey',
      {
        body,
        params: { path: { type } },
        ...options,
      }
    )

    return transformResponse(resp)
  }
}

/**
 * Stripe App
 */
export class AppStripe {
  constructor(private client: Client<paths, `${string}/${string}`>) { }

  /**
   * Create a checkout session
   * @param body - The body of the request
   * @param signal - An optional abort signal
   * @returns The checkout session
   */
  public async createCheckoutSession(
    body: CreateStripeCheckoutSessionRequest,
    options?: RequestOptions
  ) {
    const resp = await this.client.POST('/api/v1/stripe/checkout/sessions', {
      body,
      ...options,
    })

    return transformResponse(resp)
  }
}
