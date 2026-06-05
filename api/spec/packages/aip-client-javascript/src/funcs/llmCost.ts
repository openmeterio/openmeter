import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { encodePath, toURLSearchParams, encodeSort } from '../lib/encodings.js'
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
  const searchParams = toURLSearchParams({
    filter: req.filter,
    sort: encodeSort(req.sort),
    page: req.page,
  })
  return request(() =>
    http(client)
      .get('openmeter/llm-cost/prices', { ...options, searchParams })
      .json<ListLlmCostPricesResponse>(),
  )
}

export function getLlmCostPrice(
  client: Client,
  req: GetLlmCostPriceRequest,
  options?: RequestOptions,
): Promise<Result<GetLlmCostPriceResponse>> {
  const path = encodePath('openmeter/llm-cost/prices/{priceId}', {
    priceId: req.priceId,
  })
  return request(() =>
    http(client).get(path, options).json<GetLlmCostPriceResponse>(),
  )
}

export function listLlmCostOverrides(
  client: Client,
  req: ListLlmCostOverridesRequest = {},
  options?: RequestOptions,
): Promise<Result<ListLlmCostOverridesResponse>> {
  const searchParams = toURLSearchParams({
    filter: req.filter,
    page: req.page,
  })
  return request(() =>
    http(client)
      .get('openmeter/llm-cost/overrides', { ...options, searchParams })
      .json<ListLlmCostOverridesResponse>(),
  )
}

export function createLlmCostOverride(
  client: Client,
  req: CreateLlmCostOverrideRequest,
  options?: RequestOptions,
): Promise<Result<CreateLlmCostOverrideResponse>> {
  return request(() =>
    http(client)
      .post('openmeter/llm-cost/overrides', { ...options, json: req })
      .json<CreateLlmCostOverrideResponse>(),
  )
}

export function deleteLlmCostOverride(
  client: Client,
  req: DeleteLlmCostOverrideRequest,
  options?: RequestOptions,
): Promise<Result<DeleteLlmCostOverrideResponse>> {
  const path = encodePath('openmeter/llm-cost/overrides/{priceId}', {
    priceId: req.priceId,
  })
  return request(async () => {
    await http(client).delete(path, options)
  })
}
