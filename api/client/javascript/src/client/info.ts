import type { Client } from 'openapi-fetch'
import type { RequestOptions } from './common.js'
import type { operations, paths } from './schemas.js'
import { transformResponse } from './utils.js'

/**
 * Info utilities for OpenMeter
 */
export class Info {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * List supported currencies
   * @description List all supported currencies.
   * @param options - The request options
   * @returns The supported currencies
   */
  public async listCurrencies(options?: RequestOptions) {
    const resp = await this.client.GET('/api/v1/info/currencies', {
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get progress
   * @param id - The ID of the progress to get
   * @param options - The request options
   * @returns The progress
   */
  public async getProgress(
    id: operations['getProgress']['parameters']['path']['id'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/info/progress/{id}', {
      params: {
        path: { id },
      },
      ...options,
    })

    return transformResponse(resp)
  }
}
