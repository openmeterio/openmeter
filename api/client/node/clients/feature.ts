import { components, operations } from '../schemas/openapi.js'
import { RequestOptions, BaseClient, OpenMeterConfig } from './client.js'
import { Paginated } from './pagination.js'

export type Feature = components['schemas']['Feature']
export type FeatureCreateInputs = components['schemas']['FeatureCreateInputs']
export type ListFeatureQueryParams =
  operations['listFeatures']['parameters']['query']

export class FeatureClient extends BaseClient {
  constructor(config: OpenMeterConfig) {
    super(config)
  }

  /**
   * Features are the building blocks of your entitlements, part of your product offering.
   * @description
   * Features are either metered or static. A feature is metered if meterSlug is provided at creation. For metered features you can pass additional filters that will be applied when calculating feature usage, based on the meter's groupBy fields. Only meters with SUM and COUNT aggregation are supported for features.
   *
   * Features cannot be updated later, only archived.
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
   * Get Feature
   *
   * @example
   * const feature = await openmeter.features.get('ai_tokens')
   */
  public async get(id: string, options?: RequestOptions): Promise<Feature> {
    return await this.request({
      path: `/api/v1/features/${id}`,
      method: 'GET',
      options,
    })
  }

  /**
   * List features
   * @example
   * const feature = await openmeter.features.list()
   */
  public async list(
    params?: ListFeatureQueryParams,
    options?: RequestOptions
  ): Promise<Feature[] | Paginated<Feature>> {
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
   * Archive a feature
   * @description
   * Once a feature is archived it cannot be unarchived. If a feature is archived, new entitlements cannot be created for it, but archiving the feature does not affect existing entitlements.
   * @example
   * await openmeter.delete('ai_tokens')
   */
  public async delete(id: string, options?: RequestOptions): Promise<void> {
    return await this.request({
      path: `/api/v1/features/${id}`,
      method: 'DELETE',
      options,
    })
  }
}
