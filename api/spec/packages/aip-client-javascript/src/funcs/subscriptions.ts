import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { encodePath, toURLSearchParams, encodeSort } from '../lib/encodings.js'
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
  ListSubscriptionAddonsRequest,
  ListSubscriptionAddonsResponse,
  GetSubscriptionAddonRequest,
  GetSubscriptionAddonResponse,
} from '../models/operations/subscriptions.js'

export function createSubscription(
  client: Client,
  req: CreateSubscriptionRequest,
  options?: RequestOptions,
): Promise<Result<CreateSubscriptionResponse>> {
  return request(() =>
    http(client)
      .post('openmeter/subscriptions', { ...options, json: req })
      .json<CreateSubscriptionResponse>(),
  )
}

export function listSubscriptions(
  client: Client,
  req: ListSubscriptionsRequest = {},
  options?: RequestOptions,
): Promise<Result<ListSubscriptionsResponse>> {
  const searchParams = toURLSearchParams({
    page: req.page,
    sort: encodeSort(req.sort),
    filter: req.filter,
  })
  return request(() =>
    http(client)
      .get('openmeter/subscriptions', { ...options, searchParams })
      .json<ListSubscriptionsResponse>(),
  )
}

export function getSubscription(
  client: Client,
  req: GetSubscriptionRequest,
  options?: RequestOptions,
): Promise<Result<GetSubscriptionResponse>> {
  const path = encodePath('openmeter/subscriptions/{subscriptionId}', { subscriptionId: req.subscriptionId })
  return request(() =>
    http(client)
      .get(path, options)
      .json<GetSubscriptionResponse>(),
  )
}

export function cancelSubscription(
  client: Client,
  req: CancelSubscriptionRequest,
  options?: RequestOptions,
): Promise<Result<CancelSubscriptionResponse>> {
  const path = encodePath('openmeter/subscriptions/{subscriptionId}/cancel', { subscriptionId: req.subscriptionId })
  return request(() =>
    http(client)
      .post(path, { ...options, json: req.body })
      .json<CancelSubscriptionResponse>(),
  )
}

export function unscheduleCancelation(
  client: Client,
  req: UnscheduleCancelationRequest,
  options?: RequestOptions,
): Promise<Result<UnscheduleCancelationResponse>> {
  const path = encodePath('openmeter/subscriptions/{subscriptionId}/unschedule-cancelation', { subscriptionId: req.subscriptionId })
  return request(() =>
    http(client)
      .post(path, options)
      .json<UnscheduleCancelationResponse>(),
  )
}

export function changeSubscription(
  client: Client,
  req: ChangeSubscriptionRequest,
  options?: RequestOptions,
): Promise<Result<ChangeSubscriptionResponse>> {
  const path = encodePath('openmeter/subscriptions/{subscriptionId}/change', { subscriptionId: req.subscriptionId })
  return request(() =>
    http(client)
      .post(path, { ...options, json: req.body })
      .json<ChangeSubscriptionResponse>(),
  )
}

export function listSubscriptionAddons(
  client: Client,
  req: ListSubscriptionAddonsRequest,
  options?: RequestOptions,
): Promise<Result<ListSubscriptionAddonsResponse>> {
  const searchParams = toURLSearchParams({
    page: req.page,
    sort: encodeSort(req.sort),
  })
  const path = encodePath('openmeter/subscriptions/{subscriptionId}/addons', { subscriptionId: req.subscriptionId })
  return request(() =>
    http(client)
      .get(path, { ...options, searchParams })
      .json<ListSubscriptionAddonsResponse>(),
  )
}

export function getSubscriptionAddon(
  client: Client,
  req: GetSubscriptionAddonRequest,
  options?: RequestOptions,
): Promise<Result<GetSubscriptionAddonResponse>> {
  const path = encodePath('openmeter/subscriptions/{subscriptionId}/addons/{subscriptionAddonId}', { subscriptionId: req.subscriptionId, subscriptionAddonId: req.subscriptionAddonId })
  return request(() =>
    http(client)
      .get(path, options)
      .json<GetSubscriptionAddonResponse>(),
  )
}
