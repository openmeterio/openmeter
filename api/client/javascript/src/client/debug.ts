import type { Client } from 'openapi-fetch'
import type { RequestOptions } from './common.js'
import type { paths } from './schemas.js'
import { transformResponse } from './utils.js'

/**
 * Debug utilities for OpenMeter
 */
export class Debug {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Get event metrics
   * @description Returns debug metrics (in OpenMetrics format) like the number of ingested events since mindnight UTC.
   * @param options - The request options
   * @returns The debug metrics
   */
  public async getMetrics(options?: RequestOptions) {
    const resp = await this.client.GET('/api/v1/debug/metrics', {
      ...options,
    })

    return transformResponse(resp)
  }
}
