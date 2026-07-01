import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { toURLSearchParams, encodeSort } from '../lib/encodings.js'
import { toWire, fromWire, assertValid, toSnakeCase } from '../lib/wire.js'
import * as schemas from '../models/schemas.js'
import type {
  ListLlmCostPricesRequest,
  ListLlmCostPricesResponse,
  GetLlmCostPriceRequest,
  GetLlmCostPriceResponse,
  ListLlmCostOverridesRequest,
  ListLlmCostOverridesResponse,
  CreateLlmCostOverrideRequest,
  CreateLlmCostOverrideResponse,
  DeleteLlmCostOverrideRequest,
  DeleteLlmCostOverrideResponse,
} from '../models/operations/llmCost.js'

export function listLlmCostPrices(
  client: Client,
  req: ListLlmCostPricesRequest = {},
  options?: RequestOptions,
): Promise<Result<ListLlmCostPricesResponse>> {
  const searchParams = toURLSearchParams(
    toWire(
      {
        filter: req.filter,
        sort: encodeSort(req.sort, toSnakeCase),
        page: req.page,
      },
      schemas.listLlmCostPricesQueryParams,
    ),
  )
  return request(() =>
    http(client)
      .get('openmeter/llm-cost/prices', { ...options, searchParams })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.listLlmCostPricesResponseWire, data)
        }
        return fromWire(data, schemas.listLlmCostPricesResponse)
      }),
  )
}

export function getLlmCostPrice(
  client: Client,
  req: GetLlmCostPriceRequest,
  options?: RequestOptions,
): Promise<Result<GetLlmCostPriceResponse>> {
  const path = `openmeter/llm-cost/prices/${encodeURIComponent(String(req.priceId))}`
  return request(() =>
    http(client)
      .get(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.getLlmCostPriceResponseWire, data)
        }
        return fromWire(data, schemas.getLlmCostPriceResponse)
      }),
  )
}

export function listLlmCostOverrides(
  client: Client,
  req: ListLlmCostOverridesRequest = {},
  options?: RequestOptions,
): Promise<Result<ListLlmCostOverridesResponse>> {
  const searchParams = toURLSearchParams(
    toWire(
      {
        filter: req.filter,
        page: req.page,
      },
      schemas.listLlmCostOverridesQueryParams,
    ),
  )
  return request(() =>
    http(client)
      .get('openmeter/llm-cost/overrides', { ...options, searchParams })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.listLlmCostOverridesResponseWire, data)
        }
        return fromWire(data, schemas.listLlmCostOverridesResponse)
      }),
  )
}

export function createLlmCostOverride(
  client: Client,
  req: CreateLlmCostOverrideRequest,
  options?: RequestOptions,
): Promise<Result<CreateLlmCostOverrideResponse>> {
  return request(() => {
    const body = toWire(req, schemas.createLlmCostOverrideBody)
    if (client._options.validate) {
      assertValid(schemas.createLlmCostOverrideBodyWire, body)
    }
    return http(client)
      .post('openmeter/llm-cost/overrides', { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.createLlmCostOverrideResponseWire, data)
        }
        return fromWire(data, schemas.createLlmCostOverrideResponse)
      })
  })
}

export function deleteLlmCostOverride(
  client: Client,
  req: DeleteLlmCostOverrideRequest,
  options?: RequestOptions,
): Promise<Result<DeleteLlmCostOverrideResponse>> {
  const path = `openmeter/llm-cost/overrides/${encodeURIComponent(String(req.priceId))}`
  return request(async () => {
    await http(client).delete(path, options)
  })
}
