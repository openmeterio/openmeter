import { type Client } from '../core.js'
import { unwrap, type RequestOptions } from '../lib/types.js'
import {
  createSubscription,
  listSubscriptions,
  getSubscription,
  cancelSubscription,
  unscheduleCancelation,
  changeSubscription,
  createSubscriptionAddon,
  listSubscriptionAddons,
  getSubscriptionAddon,
} from '../funcs/subscriptions.js'
import type {
  CreateSubscriptionRequest,
  CreateSubscriptionResponse,
  ListSubscriptionsRequest,
  ListSubscriptionsResponse,
  GetSubscriptionRequest,
  GetSubscriptionResponse,
  CancelSubscriptionRequest,
  CancelSubscriptionResponse,
  UnscheduleCancelationRequest,
  UnscheduleCancelationResponse,
  ChangeSubscriptionRequest,
  ChangeSubscriptionResponse,
  CreateSubscriptionAddonRequest,
  CreateSubscriptionAddonResponse,
  ListSubscriptionAddonsRequest,
  ListSubscriptionAddonsResponse,
  GetSubscriptionAddonRequest,
  GetSubscriptionAddonResponse,
} from '../models/operations/subscriptions.js'

export class Subscriptions {
  constructor(private readonly _client: Client) {}

  async create(
    request: CreateSubscriptionRequest,
    options?: RequestOptions,
  ): Promise<CreateSubscriptionResponse> {
    return unwrap(await createSubscription(this._client, request, options))
  }

  async list(
    request?: ListSubscriptionsRequest,
    options?: RequestOptions,
  ): Promise<ListSubscriptionsResponse> {
    return unwrap(await listSubscriptions(this._client, request, options))
  }

  async get(
    request: GetSubscriptionRequest,
    options?: RequestOptions,
  ): Promise<GetSubscriptionResponse> {
    return unwrap(await getSubscription(this._client, request, options))
  }

  async cancel(
    request: CancelSubscriptionRequest,
    options?: RequestOptions,
  ): Promise<CancelSubscriptionResponse> {
    return unwrap(await cancelSubscription(this._client, request, options))
  }

  async unscheduleCancelation(
    request: UnscheduleCancelationRequest,
    options?: RequestOptions,
  ): Promise<UnscheduleCancelationResponse> {
    return unwrap(await unscheduleCancelation(this._client, request, options))
  }

  async change(
    request: ChangeSubscriptionRequest,
    options?: RequestOptions,
  ): Promise<ChangeSubscriptionResponse> {
    return unwrap(await changeSubscription(this._client, request, options))
  }

  async createAddon(
    request: CreateSubscriptionAddonRequest,
    options?: RequestOptions,
  ): Promise<CreateSubscriptionAddonResponse> {
    return unwrap(await createSubscriptionAddon(this._client, request, options))
  }

  async listAddons(
    request: ListSubscriptionAddonsRequest,
    options?: RequestOptions,
  ): Promise<ListSubscriptionAddonsResponse> {
    return unwrap(await listSubscriptionAddons(this._client, request, options))
  }

  async getAddon(
    request: GetSubscriptionAddonRequest,
    options?: RequestOptions,
  ): Promise<GetSubscriptionAddonResponse> {
    return unwrap(await getSubscriptionAddon(this._client, request, options))
  }
}
