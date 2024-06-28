import { components, operations } from '../schemas/openapi.js'
import { RequestOptions, BaseClient, OpenMeterConfig } from './client.js'

export type EntitlementGrant = components['schemas']['EntitlementGrant']
export type EntitlementGrantCreateInput =
  components['schemas']['EntitlementGrantCreateInput']
export type ListGrantQueryParams =
  operations['listGrants']['parameters']['query']
export type ListEntitlementGrantQueryParams =
  operations['listEntitlementGrants']['parameters']['query']

export class GrantClient extends BaseClient {
  constructor(config: OpenMeterConfig) {
    super(config)
  }

  /**
   * List grants
   * @example
   * const grant = await openmeter.grants.list()
   */
  public async list(
    params?: ListGrantQueryParams,
    options?: RequestOptions
  ): Promise<EntitlementGrant[]> {
    const searchParams = params
      ? BaseClient.toURLSearchParams(params)
      : undefined
    return await this.request({
      path: '/api/v1/grants',
      method: 'GET',
      searchParams,
      options,
    })
  }
}
