import type { Client } from 'openapi-fetch'
import type { RequestOptions } from './common.js'
import type {
  Feature,
  FeatureCreateInputs,
  operations,
  paths,
} from './schemas.js'
import { transformResponse } from './utils.js'

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
  public async create(feature: FeatureCreateInputs, options?: RequestOptions) {
    const resp = await this.client.POST('/api/v1/features', {
      body: feature,
      ...options,
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
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/features/{featureId}', {
      params: {
        path: {
          featureId: id,
        },
      },
      ...options,
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
    query?: Omit<
      operations['listFeatures']['parameters']['query'],
      'page' | 'pageSize'
    >,
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/features', {
      params: {
        query,
      },
      ...options,
    })

    return transformResponse(resp) as Feature[]
  }

  /**
   * Delete a feature by ID
   * @param id - The ID of the feature
   * @param signal - An optional abort signal
   * @returns The deleted feature
   */
  public async delete(
    id: operations['deleteFeature']['parameters']['path']['featureId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.DELETE('/api/v1/features/{featureId}', {
      params: {
        path: {
          featureId: id,
        },
      },
      ...options,
    })

    return transformResponse(resp)
  }
}
