import type { Client } from 'openapi-fetch'
import type { RequestOptions } from '../client/common.js'
import { transformResponse } from '../client/utils.js'
import type {
  BillingSubscriptionCancel,
  BillingSubscriptionChange,
  BillingSubscriptionCreate,
  operations,
  paths,
} from './schemas.js'

/**
 * Subscriptions (v3)
 *
 * Thin wrapper over the v3 subscriptions endpoints. Bodies use the v3 wire
 * shape verbatim (snake_case); no field renaming (Option A).
 */
export class Subscriptions {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Create a subscription
   */
  public async create(
    subscription: BillingSubscriptionCreate,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST('/openmeter/subscriptions', {
      body: subscription,
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List subscriptions
   */
  public async list(
    params?: operations['list-subscriptions']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/openmeter/subscriptions', {
      params: { query: params },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get a subscription by ID
   */
  public async get(subscriptionId: string, options?: RequestOptions) {
    const resp = await this.client.GET(
      '/openmeter/subscriptions/{subscriptionId}',
      {
        params: { path: { subscriptionId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Cancel a subscription
   */
  public async cancel(
    subscriptionId: string,
    body: BillingSubscriptionCancel,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/openmeter/subscriptions/{subscriptionId}/cancel',
      {
        body,
        params: { path: { subscriptionId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Change a subscription (cancel + create in one step)
   */
  public async change(
    subscriptionId: string,
    body: BillingSubscriptionChange,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/openmeter/subscriptions/{subscriptionId}/change',
      {
        body,
        params: { path: { subscriptionId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Unschedule a previously scheduled cancelation
   */
  public async unscheduleCancelation(
    subscriptionId: string,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/openmeter/subscriptions/{subscriptionId}/unschedule-cancelation',
      {
        params: { path: { subscriptionId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * List the add-ons of a subscription
   */
  public async listAddons(
    subscriptionId: string,
    params?: operations['list-subscription-addons']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/openmeter/subscriptions/{subscriptionId}/addons',
      {
        params: { path: { subscriptionId }, query: params },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Get an add-on association for a subscription
   */
  public async getAddon(
    subscriptionId: string,
    subscriptionAddonId: string,
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/openmeter/subscriptions/{subscriptionId}/addons/{subscriptionAddonId}',
      {
        params: { path: { subscriptionAddonId, subscriptionId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }
}
