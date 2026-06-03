import type { Client } from 'openapi-fetch'
import type { RequestOptions } from '../client/common.js'
import { transformResponse } from '../client/utils.js'
import type {
  CreateMeterRequest,
  MeterQueryRequest,
  operations,
  paths,
  UpdateMeterRequest,
} from './schemas.js'

/**
 * Meters (v3)
 *
 * Thin wrapper over the v3 meters endpoints. Bodies use the v3 wire shape
 * verbatim (snake_case); no field renaming (Option A).
 */
export class Meters {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Create a meter
   */
  public async create(meter: CreateMeterRequest, options?: RequestOptions) {
    const resp = await this.client.POST('/openmeter/meters', {
      body: meter,
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List meters
   */
  public async list(
    params?: operations['list-meters']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/openmeter/meters', {
      params: { query: params },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get a meter by ID
   */
  public async get(meterId: string, options?: RequestOptions) {
    const resp = await this.client.GET('/openmeter/meters/{meterId}', {
      params: { path: { meterId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Query a meter's usage
   */
  public async query(
    meterId: string,
    body: MeterQueryRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST('/openmeter/meters/{meterId}/query', {
      body,
      params: { path: { meterId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Update a meter
   */
  public async update(
    meterId: string,
    meter: UpdateMeterRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT('/openmeter/meters/{meterId}', {
      body: meter,
      params: { path: { meterId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Delete a meter by ID
   */
  public async delete(meterId: string, options?: RequestOptions) {
    const resp = await this.client.DELETE('/openmeter/meters/{meterId}', {
      params: { path: { meterId } },
      ...options,
    })

    return transformResponse(resp)
  }
}
