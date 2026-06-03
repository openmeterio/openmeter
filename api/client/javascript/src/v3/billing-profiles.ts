import type { Client } from 'openapi-fetch'
import type { RequestOptions } from '../client/common.js'
import { transformResponse } from '../client/utils.js'
import type {
  CreateBillingProfileRequest,
  operations,
  paths,
  UpsertBillingProfileRequest,
} from './schemas.js'

/**
 * Billing profiles (v3)
 *
 * Thin wrapper over the v3 billing profile endpoints. Bodies use the v3 wire
 * shape verbatim (snake_case); no field renaming (Option A).
 */
export class BillingProfiles {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Create a billing profile
   */
  public async create(
    body: CreateBillingProfileRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST('/openmeter/profiles', {
      body,
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List billing profiles
   */
  public async list(
    params?: operations['list-billing-profiles']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/openmeter/profiles', {
      params: { query: params },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get a billing profile by ID
   */
  public async get(id: string, options?: RequestOptions) {
    const resp = await this.client.GET('/openmeter/profiles/{id}', {
      params: { path: { id } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Update a billing profile
   */
  public async update(
    id: string,
    body: UpsertBillingProfileRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT('/openmeter/profiles/{id}', {
      body,
      params: { path: { id } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Delete a billing profile
   */
  public async delete(id: string, options?: RequestOptions) {
    const resp = await this.client.DELETE('/openmeter/profiles/{id}', {
      params: { path: { id } },
      ...options,
    })

    return transformResponse(resp)
  }
}
