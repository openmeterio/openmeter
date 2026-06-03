import type { Client } from 'openapi-fetch'
import type { RequestOptions } from '../client/common.js'
import { transformResponse } from '../client/utils.js'
import type { operations, paths } from './schemas.js'

/**
 * Apps (v3)
 *
 * Thin wrapper over the v3 app endpoints. Responses use the v3 wire shape
 * verbatim (snake_case); no field renaming (Option A).
 */
export class Apps {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Get an installed app by ID
   */
  public async get(appId: string, options?: RequestOptions) {
    const resp = await this.client.GET('/openmeter/apps/{appId}', {
      params: { path: { appId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List installed apps
   */
  public async list(
    params?: operations['list-apps']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/openmeter/apps', {
      params: { query: params },
      ...options,
    })

    return transformResponse(resp)
  }
}
