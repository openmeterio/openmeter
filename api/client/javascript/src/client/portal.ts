import type { Client } from 'openapi-fetch'
import type { RequestOptions } from './common.js'
import type { operations, PortalToken, paths } from './schemas.js'
import { transformResponse } from './utils.js'

/**
 * Portal
 * Manage portal tokens.
 */
export class Portal {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Create a consumer portal token
   * @param request - The request body
   * @param options - The request options
   * @returns The portal token
   */
  public async create(body: PortalToken, options?: RequestOptions) {
    const resp = await this.client.POST('/api/v1/portal/tokens', {
      body,
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List consumer portal tokens
   * @param query - The query parameters
   * @param options - The request options
   * @returns The portal tokens
   */
  public async list(
    query?: operations['listPortalTokens']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/portal/tokens', {
      params: { query },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Invalidate consumer portal tokens
   * @param body - The id or subject to invalidate
   * @param options - The request options
   * @returns The portal token
   */
  public async invalidate(
    body: operations['invalidatePortalTokens']['requestBody']['content']['application/json'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST('/api/v1/portal/tokens/invalidate', {
      body,
      ...options,
    })

    return transformResponse(resp)
  }
}
