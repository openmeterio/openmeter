import { components, operations } from '../schemas/openapi.js'
import { RequestOptions, BaseClient, OpenMeterConfig } from './client.js'

export type Entitlement =
  | EntitlementMetered
  | EntitlementStatic
  | EntitlementBoolean
export type EntitlementMetered = components['schemas']['EntitlementMetered']
export type EntitlementStatic = components['schemas']['EntitlementStatic']
export type EntitlementBoolean = components['schemas']['EntitlementBoolean']
export type RecurringPeriodEnum = components['schemas']['RecurringPeriodEnum']
export type EntitlementValue = components['schemas']['EntitlementValue']
export type WindowedBalanceHistory =
  components['schemas']['WindowedBalanceHistory']
export type EntitlementCreateInputs =
  | EntitlementMeteredCreateInputs
  | EntitlementStaticCreateInputs
  | EntitlementBooleanCreateInputs
export type EntitlementMeteredCreateInputs =
  components['schemas']['EntitlementMeteredCreateInputs']
export type EntitlementStaticCreateInputs =
  components['schemas']['EntitlementStaticCreateInputs']
export type EntitlementBooleanCreateInputs =
  components['schemas']['EntitlementBooleanCreateInputs']
export type EntitlementResetInputs =
  operations['resetEntitlementUsage']['requestBody']['content']['application/json']
export type ListEntitlementQueryParams =
  operations['listEntitlements']['parameters']['query']
export type GetEntitlementHistoryQueryParams =
  operations['getEntitlementHistory']['parameters']['query']

export class EntitlementClient extends BaseClient {
  constructor(config: OpenMeterConfig) {
    super(config)
  }

  /**
   * List all entitlements regardless of subject.
   * @description
   * Most useful for administrative purposes.
   * @example
   * const entitlement = await openmeter.entitlements.list()
   */
  public async list(
    params?: ListEntitlementQueryParams,
    options?: RequestOptions
  ): Promise<Entitlement[]> {
    const searchParams = params
      ? BaseClient.toURLSearchParams(params)
      : undefined
    return await this.request({
      path: '/api/v1/entitlements',
      method: 'GET',
      searchParams,
      options,
    })
  }
}
