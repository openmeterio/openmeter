import { components, operations } from '../schemas/openapi.js'
import { RequestOptions, BaseClient, OpenMeterConfig } from './client.js'
import { Paginated } from './pagination.js'

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
  ): Promise<EntitlementGrant[] | Paginated<EntitlementGrant>> {
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

  /**
   * Void a grant
   * @description
   * Voiding a grant means it is no longer valid, it doesn't take part in further balance calculations. Voiding a grant does not retroactively take effect, meaning any usage that has already been attributed to the grant will remain, but future usage cannot be burnt down from the grant.
   * @example
   * const grant = await openmeter.grants.list()
   */
  public async delete(
    id: string,
    options?: RequestOptions
  ): Promise<EntitlementGrant[]> {
    return await this.request({
      method: 'DELETE',
      path: `/api/v1/grants/${id}`,
      options,
    })
  }
}
