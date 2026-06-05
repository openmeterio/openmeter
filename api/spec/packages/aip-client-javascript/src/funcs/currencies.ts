import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { encodePath, toURLSearchParams, encodeSort } from '../lib/encodings.js'
import type {
  ListCurrenciesRequest,
  ListCurrenciesResponse,
  CreateCustomCurrencyRequest,
  CreateCustomCurrencyResponse,
  ListCostBasesRequest,
  ListCostBasesResponse,
  CreateCostBasisRequest,
  CreateCostBasisResponse,
} from '../models/operations/currencies.js'

export function listCurrencies(
  client: Client,
  req: ListCurrenciesRequest = {},
  options?: RequestOptions,
): Promise<Result<ListCurrenciesResponse>> {
  const searchParams = toURLSearchParams({
    page: req.page,
    sort: encodeSort(req.sort),
    filter: req.filter,
  })
  return request(() =>
    http(client)
      .get('openmeter/currencies', { ...options, searchParams })
      .json<ListCurrenciesResponse>(),
  )
}

export function createCustomCurrency(
  client: Client,
  req: CreateCustomCurrencyRequest,
  options?: RequestOptions,
): Promise<Result<CreateCustomCurrencyResponse>> {
  return request(() =>
    http(client)
      .post('openmeter/currencies/custom', { ...options, json: req })
      .json<CreateCustomCurrencyResponse>(),
  )
}

export function listCostBases(
  client: Client,
  req: ListCostBasesRequest,
  options?: RequestOptions,
): Promise<Result<ListCostBasesResponse>> {
  const searchParams = toURLSearchParams({
    filter: req.filter,
    page: req.page,
  })
  const path = encodePath('openmeter/currencies/custom/{currencyId}/cost-bases', { currencyId: req.currencyId })
  return request(() =>
    http(client)
      .get(path, { ...options, searchParams })
      .json<ListCostBasesResponse>(),
  )
}

export function createCostBasis(
  client: Client,
  req: CreateCostBasisRequest,
  options?: RequestOptions,
): Promise<Result<CreateCostBasisResponse>> {
  const path = encodePath('openmeter/currencies/custom/{currencyId}/cost-bases', { currencyId: req.currencyId })
  return request(() =>
    http(client)
      .post(path, { ...options, json: req.body })
      .json<CreateCostBasisResponse>(),
  )
}
