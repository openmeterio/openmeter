import type { Client } from 'openapi-fetch'
import type { RequestOptions } from '../client/common.js'
import { transformResponse } from '../client/utils.js'
import type {
  CreateAddonRequest,
  operations,
  paths,
  UpsertAddonRequest,
} from './schemas.js'

/**
 * Addons (v3)
 *
 * Thin wrapper over the v3 add-on endpoints. Request/response bodies use the v3
 * wire shape verbatim (snake_case); no field renaming (Option A).
 */
export class Addons {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Create an add-on
   */
  public async create(addon: CreateAddonRequest, options?: RequestOptions) {
    const resp = await this.client.POST('/openmeter/addons', {
      body: addon,
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List add-ons
   */
  public async list(
    params?: operations['list-addons']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/openmeter/addons', {
      params: { query: params },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get an add-on by ID
   */
  public async get(addonId: string, options?: RequestOptions) {
    const resp = await this.client.GET('/openmeter/addons/{addonId}', {
      params: { path: { addonId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Update (replace) an add-on
   */
  public async update(
    addonId: string,
    addon: UpsertAddonRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT('/openmeter/addons/{addonId}', {
      body: addon,
      params: { path: { addonId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Delete an add-on
   */
  public async delete(addonId: string, options?: RequestOptions) {
    const resp = await this.client.DELETE('/openmeter/addons/{addonId}', {
      params: { path: { addonId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Publish an add-on version
   */
  public async publish(addonId: string, options?: RequestOptions) {
    const resp = await this.client.POST('/openmeter/addons/{addonId}/publish', {
      params: { path: { addonId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Archive an add-on version
   */
  public async archive(addonId: string, options?: RequestOptions) {
    const resp = await this.client.POST('/openmeter/addons/{addonId}/archive', {
      params: { path: { addonId } },
      ...options,
    })

    return transformResponse(resp)
  }
}
