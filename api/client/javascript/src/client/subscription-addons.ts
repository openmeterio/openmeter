import type { Client } from 'openapi-fetch'
import type { RequestOptions } from './common.js'
import type { operations, paths } from './schemas.js'
import { transformResponse } from './utils.js'

export class SubscriptionAddons {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Create a new subscription addon
   * @param subscriptionId - The ID of the subscription
   * @param addon - The subscription addon to create
   * @param options - Optional request options
   * @returns The created subscription addon
   */
  public async create(
    subscriptionId: string,
    addon: operations['createSubscriptionAddon']['requestBody']['content']['application/json'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/api/v1/subscriptions/{subscriptionId}/addons',
      {
        body: addon,
        params: { path: { subscriptionId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * List all addons of a subscription
   * @param subscriptionId - The ID of the subscription
   * @param options - Optional request options
   * @returns A list of subscription addons
   */
  public async list(subscriptionId: string, options?: RequestOptions) {
    const resp = await this.client.GET(
      '/api/v1/subscriptions/{subscriptionId}/addons',
      {
        params: { path: { subscriptionId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Get a subscription addon by id
   * @param subscriptionId - The ID of the subscription
   * @param subscriptionAddonId - The ID of the subscription addon
   * @param options - Optional request options
   * @returns The subscription addon
   */
  public async get(
    subscriptionId: string,
    subscriptionAddonId: string,
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/api/v1/subscriptions/{subscriptionId}/addons/{subscriptionAddonId}',
      {
        params: { path: { subscriptionAddonId, subscriptionId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Updates a subscription addon
   * @param subscriptionId - The ID of the subscription
   * @param subscriptionAddonId - The ID of the subscription addon to update
   * @param addon - The subscription addon data to update
   * @param options - Optional request options
   * @returns The updated subscription addon
   */
  public async update(
    subscriptionId: string,
    subscriptionAddonId: string,
    addon: operations['updateSubscriptionAddon']['requestBody']['content']['application/json'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.PATCH(
      '/api/v1/subscriptions/{subscriptionId}/addons/{subscriptionAddonId}',
      {
        body: addon,
        params: { path: { subscriptionAddonId, subscriptionId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }
}
