import { components, operations } from '../schemas/openapi.js'
import { RequestOptions, BaseClient, OpenMeterConfig } from './client.js'

export type Feature = components['schemas']['Feature']
export type FeatureCreateInputs = components['schemas']['FeatureCreateInputs']
export type ListFeatureQueryParams = operations['listFeatures']['parameters']['query']

export class FeatureClient extends BaseClient {
  constructor(config: OpenMeterConfig) {
    super(config)
  }

  /**
   * Create Feature
   * Features are the building blocks of your entitlements, part of your product offering.
   *
   * @example
   * const feature = await openmeter.features.create({
   *  key: 'ai_tokens',
   *  name: 'AI Tokens',
   *  // optional
   *  meterSlug: 'tokens_total',
   * })
   */
  public async create(
    input: FeatureCreateInputs,
    options?: RequestOptions
  ): Promise<Feature> {
    return await this.request({
      path: '/api/v1/features',
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(input),
      options,
    })
  }

  /**
   * Get Feature by ID or Key
   *
   * @example
   * const feature = await openmeter.features.get('ai_tokens')
   */
  public async get(idOrKey: string, options?: RequestOptions): Promise<Feature> {
    return await this.request({
      path: `/api/v1/features/${idOrKey}`,
      method: 'GET',
      options,
    })
  }

  /**
   * List features
   * @example
   * const feature = await openmeter.features.list()
  */
  public async list(params?: ListFeatureQueryParams, options?: RequestOptions): Promise<Feature[]> {
    const searchParams = params
      ? BaseClient.toURLSearchParams(params)
      : undefined
    return await this.request({
      path: '/api/v1/features',
      method: 'GET',
      searchParams,
      options,
    })
  }

  /**
   * Delete feature by ID or Key
   * @example
   * const feature = await openmeter.delete('ai_tokens)
  */
  public async delete(
    idOrKey: string,
    options?: RequestOptions
  ): Promise<void> {
    return await this.request({
      path: `/api/v1/features/${idOrKey}`,
      method: 'DELETE',
      options,
    })
  }
}

