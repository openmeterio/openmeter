import type { Client } from 'openapi-fetch'
import type { RequestOptions } from './common.js'
import type { AddonCreate, operations, paths } from './schemas.js'
import { transformResponse } from './utils.js'

export class Addons {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Create a addon
   * @param addon - The addon to create
   * @param options - Optional request options
   * @returns The created addon
   */
  public async create(addon: AddonCreate, options?: RequestOptions) {
    const resp = await this.client.POST('/api/v1/addons', {
      body: addon,
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List addons
   * @param params - Optional parameters for listing addons
   * @param options - Optional request options
   * @returns A list of addons
   */
  public async list(
    params?: operations['listAddons']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/addons', {
      params: { query: params },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get an addon by ID
   * @param addonId - The ID of the addon to retrieve
   * @param options - Optional request options
   * @returns The addon
   */
  public async get(addonId: string, options?: RequestOptions) {
    const resp = await this.client.GET('/api/v1/addons/{addonId}', {
      params: { path: { addonId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Update an addon
   * @param addonId - The ID of the addon to update
   * @param addon - The addon data to update
   * @param options - Optional request options
   * @returns The updated addon
   */
  public async update(
    addonId: string,
    addon: operations['updateAddon']['requestBody']['content']['application/json'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT('/api/v1/addons/{addonId}', {
      body: addon,
      params: { path: { addonId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Delete an addon by ID
   * @param addonId - The ID of the addon to delete
   * @param options - Optional request options
   * @returns void or standard error response structure
   */
  public async delete(addonId: string, options?: RequestOptions) {
    const resp = await this.client.DELETE('/api/v1/addons/{addonId}', {
      params: { path: { addonId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Publish an addon
   * @param addonId - The ID of the addon to publish
   * @param options - Optional request options
   * @returns The published addon
   */
  public async publish(addonId: string, options?: RequestOptions) {
    const resp = await this.client.POST('/api/v1/addons/{addonId}/publish', {
      params: { path: { addonId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Archive an addon
   * @param addonId - The ID of the addon to archive
   * @param options - Optional request options
   * @returns The archived addon
   */
  public async archive(addonId: string, options?: RequestOptions) {
    const resp = await this.client.POST('/api/v1/addons/{addonId}/archive', {
      params: { path: { addonId } },
      ...options,
    })

    return transformResponse(resp)
  }
}
