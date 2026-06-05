import { type Client } from '../core.js'
import { unwrap, type RequestOptions } from '../lib/types.js'
import {
  listCurrencies,
  createCustomCurrency,
  listCostBases,
  createCostBasis,
} from '../funcs/currencies.js'
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

export class Currencies {
  constructor(private readonly _client: Client) {}

  async list(
    request?: ListCurrenciesRequest,
    options?: RequestOptions,
  ): Promise<ListCurrenciesResponse> {
    return unwrap(await listCurrencies(this._client, request, options))
  }

  async createCustomCurrency(
    request: CreateCustomCurrencyRequest,
    options?: RequestOptions,
  ): Promise<CreateCustomCurrencyResponse> {
    return unwrap(await createCustomCurrency(this._client, request, options))
  }

  async listCostBases(
    request: ListCostBasesRequest,
    options?: RequestOptions,
  ): Promise<ListCostBasesResponse> {
    return unwrap(await listCostBases(this._client, request, options))
  }

  async createCostBasis(
    request: CreateCostBasisRequest,
    options?: RequestOptions,
  ): Promise<CreateCostBasisResponse> {
    return unwrap(await createCostBasis(this._client, request, options))
  }
}
