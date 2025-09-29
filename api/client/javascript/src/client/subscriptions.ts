import type { Client } from 'openapi-fetch'
import type { RequestOptions } from './common.js'
import type {
  operations,
  paths,
  SubscriptionChange,
  SubscriptionCreate,
  SubscriptionEdit,
} from './schemas.js'
import { transformResponse } from './utils.js'

/**
 * Subscriptions
 */
export class Subscriptions {
  constructor(private readonly client: Client<paths, `${string}/${string}`>) {}

  /**
   * Create a subscription
   * @param body - The subscription to create
   * @param signal - An optional abort signal
   * @returns The created subscription
   */
  public async create(body: SubscriptionCreate, options?: RequestOptions) {
    const resp = await this.client.POST('/api/v1/subscriptions', {
      body,
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get a subscription
   * @param id - The subscription ID
   * @param signal - An optional abort signal
   * @returns The subscription
   */
  public async get(
    id: operations['getSubscription']['parameters']['path']['subscriptionId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/api/v1/subscriptions/{subscriptionId}',
      {
        params: { path: { subscriptionId: id } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Edit a subscription
   * @param id - The subscription ID
   * @param body - The subscription to edit
   * @param signal - An optional abort signal
   * @returns The edited subscription
   */
  public async edit(
    id: operations['editSubscription']['parameters']['path']['subscriptionId'],
    body: SubscriptionEdit,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PATCH(
      '/api/v1/subscriptions/{subscriptionId}',
      {
        body,
        params: { path: { subscriptionId: id } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Cancel a subscription
   * @param id - The subscription ID
   * @param body - The subscription to cancel
   * @param signal - An optional abort signal
   * @returns The canceled subscription
   */
  public async cancel(
    id: operations['cancelSubscription']['parameters']['path']['subscriptionId'],
    body: operations['cancelSubscription']['requestBody']['content']['application/json'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/api/v1/subscriptions/{subscriptionId}/cancel',
      {
        body,
        params: { path: { subscriptionId: id } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Change a subscription
   * @description Closes a running subscription and starts a new one according to the specification. Can be used for upgrades, downgrades, and plan changes.
   * @param id - The subscription ID
   * @param body - The subscription to change
   * @param signal - An optional abort signal
   * @returns The changed subscription
   */
  public async change(
    id: operations['changeSubscription']['parameters']['path']['subscriptionId'],
    body: SubscriptionChange,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/api/v1/subscriptions/{subscriptionId}/change',
      {
        body,
        params: { path: { subscriptionId: id } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Migrate a subscription
   * @description Migrates the subscripiton to the provided version of the current plan.
   * @param id - The subscription ID
   * @param body - The subscription to migrate
   * @param signal - An optional abort signal
   * @returns The migrated subscription
   */
  public async migrate(
    id: operations['migrateSubscription']['parameters']['path']['subscriptionId'],
    body: operations['migrateSubscription']['requestBody']['content']['application/json'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/api/v1/subscriptions/{subscriptionId}/migrate',
      {
        body,
        params: { path: { subscriptionId: id } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Unschedule a cancelation
   * @param id - The subscription ID
   * @param signal - An optional abort signal
   * @returns The unscheduled subscription
   */
  public async unscheduleCancelation(
    id: operations['unscheduleCancelation']['parameters']['path']['subscriptionId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/api/v1/subscriptions/{subscriptionId}/unschedule-cancelation',
      {
        params: { path: { subscriptionId: id } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Delete subscription
   * @description Deletes a subscription. Only scheduled subscriptions can be deleted.
   * @param subscriptionId - The ID of the subscription to delete
   * @param options - Optional request options
   * @returns void or standard error response structure
   */
  public async delete(
    subscriptionId: operations['deleteSubscription']['parameters']['path']['subscriptionId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.DELETE(
      '/api/v1/subscriptions/{subscriptionId}',
      {
        params: {
          path: { subscriptionId },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }
}
