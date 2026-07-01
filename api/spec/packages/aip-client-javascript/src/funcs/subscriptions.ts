import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { toURLSearchParams, encodeSort } from '../lib/encodings.js'
import { toWire, fromWire, assertValid, toSnakeCase } from '../lib/wire.js'
import * as schemas from '../models/schemas.js'
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

export function createSubscription(
  client: Client,
  req: CreateSubscriptionRequest,
  options?: RequestOptions,
): Promise<Result<CreateSubscriptionResponse>> {
  return request(() => {
    const body = toWire(req, schemas.createSubscriptionBody)
    if (client._options.validate) {
      assertValid(schemas.createSubscriptionBodyWire, body)
    }
    return http(client)
      .post('openmeter/subscriptions', { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.createSubscriptionResponseWire, data)
        }
        return fromWire(data, schemas.createSubscriptionResponse)
      })
  })
}

export function listSubscriptions(
  client: Client,
  req: ListSubscriptionsRequest = {},
  options?: RequestOptions,
): Promise<Result<ListSubscriptionsResponse>> {
  const searchParams = toURLSearchParams(
    toWire(
      {
        page: req.page,
        sort: encodeSort(req.sort, toSnakeCase),
        filter: req.filter,
      },
      schemas.listSubscriptionsQueryParams,
    ),
  )
  return request(() =>
    http(client)
      .get('openmeter/subscriptions', { ...options, searchParams })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.listSubscriptionsResponseWire, data)
        }
        return fromWire(data, schemas.listSubscriptionsResponse)
      }),
  )
}

export function getSubscription(
  client: Client,
  req: GetSubscriptionRequest,
  options?: RequestOptions,
): Promise<Result<GetSubscriptionResponse>> {
  const path = `openmeter/subscriptions/${(() => {
    if (req.subscriptionId === undefined) {
      throw new Error('missing path parameter: subscriptionId')
    }
    return encodeURIComponent(String(req.subscriptionId))
  })()}`
  return request(() =>
    http(client)
      .get(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.getSubscriptionResponseWire, data)
        }
        return fromWire(data, schemas.getSubscriptionResponse)
      }),
  )
}

export function cancelSubscription(
  client: Client,
  req: CancelSubscriptionRequest,
  options?: RequestOptions,
): Promise<Result<CancelSubscriptionResponse>> {
  const path = `openmeter/subscriptions/${(() => {
    if (req.subscriptionId === undefined) {
      throw new Error('missing path parameter: subscriptionId')
    }
    return encodeURIComponent(String(req.subscriptionId))
  })()}/cancel`
  return request(() => {
    const body = toWire(req.body, schemas.cancelSubscriptionBody)
    if (client._options.validate) {
      assertValid(schemas.cancelSubscriptionBodyWire, body)
    }
    return http(client)
      .post(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.cancelSubscriptionResponseWire, data)
        }
        return fromWire(data, schemas.cancelSubscriptionResponse)
      })
  })
}

export function unscheduleCancelation(
  client: Client,
  req: UnscheduleCancelationRequest,
  options?: RequestOptions,
): Promise<Result<UnscheduleCancelationResponse>> {
  const path = `openmeter/subscriptions/${(() => {
    if (req.subscriptionId === undefined) {
      throw new Error('missing path parameter: subscriptionId')
    }
    return encodeURIComponent(String(req.subscriptionId))
  })()}/unschedule-cancelation`
  return request(() =>
    http(client)
      .post(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.unscheduleCancelationResponseWire, data)
        }
        return fromWire(data, schemas.unscheduleCancelationResponse)
      }),
  )
}

export function changeSubscription(
  client: Client,
  req: ChangeSubscriptionRequest,
  options?: RequestOptions,
): Promise<Result<ChangeSubscriptionResponse>> {
  const path = `openmeter/subscriptions/${(() => {
    if (req.subscriptionId === undefined) {
      throw new Error('missing path parameter: subscriptionId')
    }
    return encodeURIComponent(String(req.subscriptionId))
  })()}/change`
  return request(() => {
    const body = toWire(req.body, schemas.changeSubscriptionBody)
    if (client._options.validate) {
      assertValid(schemas.changeSubscriptionBodyWire, body)
    }
    return http(client)
      .post(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.changeSubscriptionResponseWire, data)
        }
        return fromWire(data, schemas.changeSubscriptionResponse)
      })
  })
}

export function createSubscriptionAddon(
  client: Client,
  req: CreateSubscriptionAddonRequest,
  options?: RequestOptions,
): Promise<Result<CreateSubscriptionAddonResponse>> {
  const path = `openmeter/subscriptions/${(() => {
    if (req.subscriptionId === undefined) {
      throw new Error('missing path parameter: subscriptionId')
    }
    return encodeURIComponent(String(req.subscriptionId))
  })()}/addons`
  return request(() => {
    const body = toWire(req.body, schemas.createSubscriptionAddonBody)
    if (client._options.validate) {
      assertValid(schemas.createSubscriptionAddonBodyWire, body)
    }
    return http(client)
      .post(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.createSubscriptionAddonResponseWire, data)
        }
        return fromWire(data, schemas.createSubscriptionAddonResponse)
      })
  })
}

export function listSubscriptionAddons(
  client: Client,
  req: ListSubscriptionAddonsRequest,
  options?: RequestOptions,
): Promise<Result<ListSubscriptionAddonsResponse>> {
  const searchParams = toURLSearchParams(
    toWire(
      {
        page: req.page,
        sort: encodeSort(req.sort, toSnakeCase),
      },
      schemas.listSubscriptionAddonsQueryParams,
    ),
  )
  const path = `openmeter/subscriptions/${(() => {
    if (req.subscriptionId === undefined) {
      throw new Error('missing path parameter: subscriptionId')
    }
    return encodeURIComponent(String(req.subscriptionId))
  })()}/addons`
  return request(() =>
    http(client)
      .get(path, { ...options, searchParams })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.listSubscriptionAddonsResponseWire, data)
        }
        return fromWire(data, schemas.listSubscriptionAddonsResponse)
      }),
  )
}

export function getSubscriptionAddon(
  client: Client,
  req: GetSubscriptionAddonRequest,
  options?: RequestOptions,
): Promise<Result<GetSubscriptionAddonResponse>> {
  const path = `openmeter/subscriptions/${(() => {
    if (req.subscriptionId === undefined) {
      throw new Error('missing path parameter: subscriptionId')
    }
    return encodeURIComponent(String(req.subscriptionId))
  })()}/addons/${(() => {
    if (req.subscriptionAddonId === undefined) {
      throw new Error('missing path parameter: subscriptionAddonId')
    }
    return encodeURIComponent(String(req.subscriptionAddonId))
  })()}`
  return request(() =>
    http(client)
      .get(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.getSubscriptionAddonResponseWire, data)
        }
        return fromWire(data, schemas.getSubscriptionAddonResponse)
      }),
  )
}
