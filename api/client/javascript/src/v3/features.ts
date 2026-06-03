import type { Client } from 'openapi-fetch'
import type { RequestOptions } from '../client/common.js'
import { transformResponse } from '../client/utils.js'
import type {
  CreateFeatureRequest,
  MeterQueryRequest,
  operations,
  paths,
  UpdateFeatureRequest,
} from './schemas.js'

/**
 * Features (v3)
 *
 * Thin wrapper over the v3 features endpoints. Bodies use the v3 wire shape
 * verbatim (snake_case); no field renaming (Option A).
 */
export class Features {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Create a feature
   */
  public async create(feature: CreateFeatureRequest, options?: RequestOptions) {
    const resp = await this.client.POST('/openmeter/features', {
      body: feature,
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List features
   */
  public async list(
    params?: operations['list-features']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/openmeter/features', {
      params: { query: params },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get a feature by ID
   */
  public async get(featureId: string, options?: RequestOptions) {
    const resp = await this.client.GET('/openmeter/features/{featureId}', {
      params: { path: { featureId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Delete a feature by ID
   */
  public async delete(featureId: string, options?: RequestOptions) {
    const resp = await this.client.DELETE('/openmeter/features/{featureId}', {
      params: { path: { featureId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Update a feature
   */
  public async update(
    featureId: string,
    feature: UpdateFeatureRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PATCH('/openmeter/features/{featureId}', {
      body: feature,
      params: { path: { featureId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Query a feature's cost
   */
  public async queryCost(
    featureId: string,
    body?: MeterQueryRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/openmeter/features/{featureId}/cost/query',
      {
        body,
        params: { path: { featureId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }
}
