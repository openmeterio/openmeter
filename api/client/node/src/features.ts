import { transformResponse } from './utils.js'
import type { FeatureCreateInputs, operations, paths } from './schemas.js'
import type { Client } from 'openapi-fetch'

/**
 * Features
 * @description Features are the building blocks of your application. They represent the capabilities or services that your application offers.
 */
export class Features {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Create a feature
   * @param feature - The feature to create
   * @param signal - An optional abort signal
   * @returns The created feature
   */
  public async create(feature: FeatureCreateInputs, signal?: AbortSignal) {
    const resp = await this.client.POST('/api/v1/features', {
      signal,
      body: feature,
    })

    return transformResponse(resp)
  }

  /**
   * Get a feature by ID
   * @param id - The ID of the feature
   * @param signal - An optional abort signal
   * @returns The feature
   */
  public async get(
    id: operations['getFeature']['parameters']['path']['featureId'],
    signal?: AbortSignal
  ) {
    const resp = await this.client.GET('/api/v1/features/{featureId}', {
      signal,
      params: {
        path: {
          featureId: id,
        },
      },
    })

    return transformResponse(resp)
  }

  /**
   * List features
   * @param query - The query parameters
   * @param signal - An optional abort signal
   * @returns The features
   */
  public async list(
    query?: operations['listFeatures']['parameters']['query'],
    signal?: AbortSignal
  ) {
    const resp = await this.client.GET('/api/v1/features', {
      signal,
      params: {
        query,
      },
    })

    return transformResponse(resp)
  }

  /**
   * Delete a feature by ID
   * @param id - The ID of the feature
   * @param signal - An optional abort signal
   * @returns The deleted feature
   */
  public async delete(
    id: operations['deleteFeature']['parameters']['path']['featureId'],
    signal?: AbortSignal
  ) {
    const resp = await this.client.DELETE('/api/v1/features/{featureId}', {
      signal,
      params: {
        path: {
          featureId: id,
        },
      },
    })

    return transformResponse(resp)
  }
}
