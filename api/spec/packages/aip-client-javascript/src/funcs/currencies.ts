import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { toURLSearchParams, encodeSort } from '../lib/encodings.js'
import { toWire, fromWire, assertValid, toSnakeCase } from '../lib/wire.js'
import * as schemas from '../models/schemas.js'
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
  const searchParams = toURLSearchParams(
    toWire(
      {
        page: req.page,
        sort: encodeSort(req.sort, toSnakeCase),
        filter: req.filter,
      },
      schemas.listCurrenciesQueryParams,
    ),
  )
  return request(() =>
    http(client)
      .get('openmeter/currencies', { ...options, searchParams })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.listCurrenciesResponseWire, data)
        }
        return fromWire(data, schemas.listCurrenciesResponse)
      }),
  )
}

export function createCustomCurrency(
  client: Client,
  req: CreateCustomCurrencyRequest,
  options?: RequestOptions,
): Promise<Result<CreateCustomCurrencyResponse>> {
  return request(() => {
    const body = toWire(req, schemas.createCustomCurrencyBody)
    if (client._options.validate) {
      assertValid(schemas.createCustomCurrencyBodyWire, body)
    }
    return http(client)
      .post('openmeter/currencies/custom', { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.createCustomCurrencyResponseWire, data)
        }
        return fromWire(data, schemas.createCustomCurrencyResponse)
      })
  })
}

export function listCostBases(
  client: Client,
  req: ListCostBasesRequest,
  options?: RequestOptions,
): Promise<Result<ListCostBasesResponse>> {
  const searchParams = toURLSearchParams(
    toWire(
      {
        filter: req.filter,
        page: req.page,
      },
      schemas.listCostBasesQueryParams,
    ),
  )
  const path = `openmeter/currencies/custom/${(() => {
    if (req.currencyId === undefined) {
      throw new Error('missing path parameter: currencyId')
    }
    return encodeURIComponent(String(req.currencyId))
  })()}/cost-bases`
  return request(() =>
    http(client)
      .get(path, { ...options, searchParams })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.listCostBasesResponseWire, data)
        }
        return fromWire(data, schemas.listCostBasesResponse)
      }),
  )
}

export function createCostBasis(
  client: Client,
  req: CreateCostBasisRequest,
  options?: RequestOptions,
): Promise<Result<CreateCostBasisResponse>> {
  const path = `openmeter/currencies/custom/${(() => {
    if (req.currencyId === undefined) {
      throw new Error('missing path parameter: currencyId')
    }
    return encodeURIComponent(String(req.currencyId))
  })()}/cost-bases`
  return request(() => {
    const body = toWire(req.body, schemas.createCostBasisBody)
    if (client._options.validate) {
      assertValid(schemas.createCostBasisBodyWire, body)
    }
    return http(client)
      .post(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.createCostBasisResponseWire, data)
        }
        return fromWire(data, schemas.createCostBasisResponse)
      })
  })
}
