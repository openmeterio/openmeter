import type { Client } from 'openapi-fetch'
import type { RequestOptions } from '../client/common.js'
import { transformResponse } from '../client/utils.js'
import type {
  CreateCostBasisRequest,
  CreateCurrencyCustomRequest,
  operations,
  paths,
} from './schemas.js'

/**
 * Currencies (v3)
 *
 * Thin wrapper over the v3 currency endpoints. Bodies use the v3 wire shape
 * verbatim (snake_case); no field renaming (Option A).
 */
export class Currencies {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Create a custom currency
   */
  public async createCustom(
    body: CreateCurrencyCustomRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST('/openmeter/currencies/custom', {
      body,
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List currencies
   */
  public async list(
    params?: operations['list-currencies']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/openmeter/currencies', {
      params: { query: params },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Create a cost basis for a custom currency
   */
  public async createCostBasis(
    currencyId: string,
    body: CreateCostBasisRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/openmeter/currencies/custom/{currencyId}/cost-bases',
      {
        body,
        params: { path: { currencyId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * List cost bases for a custom currency
   */
  public async listCostBases(
    currencyId: string,
    params?: operations['list-cost-bases']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/openmeter/currencies/custom/{currencyId}/cost-bases',
      {
        params: { path: { currencyId }, query: params },
        ...options,
      },
    )

    return transformResponse(resp)
  }
}
