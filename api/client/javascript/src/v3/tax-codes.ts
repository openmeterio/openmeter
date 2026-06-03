import type { Client } from 'openapi-fetch'
import type { RequestOptions } from '../client/common.js'
import { transformResponse } from '../client/utils.js'
import type {
  CreateTaxCodeRequest,
  operations,
  paths,
  UpsertTaxCodeRequest,
} from './schemas.js'

/**
 * Tax codes (v3)
 *
 * Thin wrapper over the v3 tax code endpoints. Bodies use the v3 wire shape
 * verbatim (snake_case); no field renaming (Option A).
 */
export class TaxCodes {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Create a tax code
   */
  public async create(body: CreateTaxCodeRequest, options?: RequestOptions) {
    const resp = await this.client.POST('/openmeter/tax-codes', {
      body,
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get a tax code by ID
   */
  public async get(taxCodeId: string, options?: RequestOptions) {
    const resp = await this.client.GET('/openmeter/tax-codes/{taxCodeId}', {
      params: { path: { taxCodeId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List tax codes
   */
  public async list(
    params?: operations['list-tax-codes']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/openmeter/tax-codes', {
      params: { query: params },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Upsert (replace) a tax code
   */
  public async upsert(
    taxCodeId: string,
    body: UpsertTaxCodeRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT('/openmeter/tax-codes/{taxCodeId}', {
      body,
      params: { path: { taxCodeId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Delete a tax code
   */
  public async delete(taxCodeId: string, options?: RequestOptions) {
    const resp = await this.client.DELETE('/openmeter/tax-codes/{taxCodeId}', {
      params: { path: { taxCodeId } },
      ...options,
    })

    return transformResponse(resp)
  }
}
