import { z } from 'zod'
import * as schemas from '../schemas.js'
import type {
  CostBasis,
  CostBasisPagePaginatedResponse,
  CreateCostBasisRequest as CreateCostBasisRequestBody,
  CreateCurrencyCustomRequest,
  CurrencyCustom,
  CurrencyPagePaginatedResponse,
  ListCostBasesParamsFilter,
  ListCurrenciesParamsFilter,
  SortQueryInput,
} from '../types.js'

export interface ListCurrenciesQuery {
  /** Determines which page of the collection to retrieve. */
  page?: { size?: number; number?: number }
  /** Sort currencies returned in the response. Supported sort attributes are: - `code` (default) - `name` The `asc` suffix is optional as the default sort order is ascending. The `desc` suffix is used to specify a descending order. */
  sort?: SortQueryInput
  /** Filter currencies returned in the response. To filter currencies by type add the following query param: filter[type]=custom */
  filter?: ListCurrenciesParamsFilter
}

export type ListCurrenciesRequest = ListCurrenciesQuery
export type ListCurrenciesResponse = CurrencyPagePaginatedResponse

export type CreateCustomCurrencyRequest = CreateCurrencyCustomRequest
export type CreateCustomCurrencyResponse = CurrencyCustom

export interface ListCostBasesQuery {
  /** Filter cost bases returned in the response. To filter cost bases by fiat currency code add the following query param: filter[fiat_code]=USD */
  filter?: ListCostBasesParamsFilter
  /** Determines which page of the collection to retrieve. */
  page?: { size?: number; number?: number }
}

export type ListCostBasesRequest = ListCostBasesQuery & { currencyId: string }
export type ListCostBasesResponse = CostBasisPagePaginatedResponse

export type CreateCostBasisRequest = {
  currencyId: string
  body: CreateCostBasisRequestBody
}
export type CreateCostBasisResponse = CostBasis
