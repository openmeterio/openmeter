import { components } from '../schemas/openapi.js'
import { RequestOptions, BaseClient, OpenMeterConfig } from './client.js'
import {
  Entitlement,
  EntitlementCreateInputs,
  EntitlementResetInputs,
  EntitlementValue,
  GetEntitlementHistoryQueryParams,
  ListEntitlementQueryParams,
  WindowedBalanceHistory,
} from './entitlement.js'

export type Subject = components['schemas']['Subject']

export class SubjectClient extends BaseClient {
  constructor(config: OpenMeterConfig) {
    super(config)
  }

  /**
   * Upsert subject
   * Useful to map display name and metadata to subjects
   * @note OpenMeter Cloud only feature
   */
  public async upsert(
    subject: Omit<Subject, 'id'>[],
    options?: RequestOptions
  ): Promise<Subject[]> {
    return await this.request({
      path: '/api/v1/subjects',
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(subject),
      options,
    })
  }

  /**
   * Get subject by id or key
   * @note OpenMeter Cloud only feature
   */
  public async get(
    idOrKey: string,
    options?: RequestOptions
  ): Promise<Subject> {
    return await this.request({
      path: `/api/v1/subjects/${idOrKey}`,
      method: 'GET',
      options,
    })
  }

  /**
   * List subjects
   * @note OpenMeter Cloud only feature
   */
  public async list(options?: RequestOptions): Promise<Subject[]> {
    return await this.request({
      path: '/api/v1/subjects',
      method: 'GET',
      options,
    })
  }

  /**
   * Delete subject by id or key
   * @note OpenMeter Cloud only feature
   */
  public async delete(
    idOrKey: string,
    options?: RequestOptions
  ): Promise<void> {
    return await this.request({
      path: `/api/v1/subjects/${idOrKey}`,
      method: 'DELETE',
      options,
    })
  }

  /** Entitlements **/

  /**
   * Create Entitlement
   * Entitlements allows you to manage subject feature access, balances, and usage limits.
   *
   * @example
   * // Issue 10,000,000 tokens every month
   * const entitlement = await openmeter.subjects.createEntitlement('customer-1', {
   *    type: 'metered',
   *    featureKey: 'ai_tokens',
   *    usagePeriod: {
   *      interval: 'MONTH',
   *    },
   *    issueAfterReset: 10000000,
   * })
   */
  public async createEntitlement(
    subjectIdOrKey: string,
    input: EntitlementCreateInputs,
    options?: RequestOptions
  ): Promise<Entitlement> {
    return await this.request({
      path: `/api/v1/subjects/${subjectIdOrKey}/entitlements`,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(input),
      options,
    })
  }

  /**
   * List entitlements
   * @example
   * const entitlement = await openmeter.subjects.listEntitlements('customer-1')
   */
  public async listEntitlements(
    subjectIdOrKey: string,
    params?: ListEntitlementQueryParams,
    options?: RequestOptions
  ): Promise<Entitlement[]> {
    const searchParams = params
      ? BaseClient.toURLSearchParams(params)
      : undefined
    return await this.request({
      path: `/api/v1/subjects/${subjectIdOrKey}/entitlements`,
      method: 'GET',
      searchParams,
      options,
    })
  }

  /**
   * Get entitlement by ID by Feature ID or by Feature Key
   *
   * @example
   * const entitlement = await openmeter.subjects.getEntitlement('customer-1', 'ai_tokens')
   */
  public async getEntitlement(
    subjectIdOrKey: string,
    entitlementIdOrFeatureIdOrFeatureKey: string,
    options?: RequestOptions
  ): Promise<Entitlement> {
    return await this.request({
      path: `/api/v1/subjects/${subjectIdOrKey}/entitlements/${entitlementIdOrFeatureIdOrFeatureKey}`,
      method: 'GET',
      options,
    })
  }

  /**
   * Delete entitlement by ID by Feature ID or by Feature Key
   * @example
   * await openmeter.deleteEntitlement('customer-1', 'ai_tokens')
   */
  public async deleteEntitlement(
    subjectIdOrKey: string,
    entitlementIdOrFeatureIdOrFeatureKey: string,
    options?: RequestOptions
  ): Promise<void> {
    return await this.request({
      path: `/api/v1/subjects/${subjectIdOrKey}/entitlements/${entitlementIdOrFeatureIdOrFeatureKey}`,
      method: 'DELETE',
      options,
    })
  }

  /**
   * Get entitlement value by ID by Feature ID or by Feature Key
   *
   * @example
   * const value = await openmeter.subjects.getEntitlementValue('customer-1', 'ai_tokens')
   */
  public async getEntitlementValue(
    subjectIdOrKey: string,
    entitlementIdOrFeatureIdOrFeatureKey: string,
    options?: RequestOptions
  ): Promise<EntitlementValue> {
    return await this.request({
      path: `/api/v1/subjects/${subjectIdOrKey}/entitlements/${entitlementIdOrFeatureIdOrFeatureKey}/value`,
      method: 'GET',
      options,
    })
  }

  /**
   * Get entitlement history by ID by Feature ID or by Feature Key
   * @example
   * const entitlement = await openmeter.subjects.getEntitlementHistory('customer-1', 'ai_tokens')
   */
  public async getEntitlementHistory(
    subjectIdOrKey: string,
    entitlementIdOrFeatureIdOrFeatureKey: string,
    params?: GetEntitlementHistoryQueryParams,
    options?: RequestOptions
  ): Promise<WindowedBalanceHistory[]> {
    const searchParams = params
      ? BaseClient.toURLSearchParams(params)
      : undefined
    return await this.request({
      path: `/api/v1/subjects/${subjectIdOrKey}/entitlements/${entitlementIdOrFeatureIdOrFeatureKey}/history`,
      method: 'GET',
      searchParams,
      options,
    })
  }

  /**
   * Reset Entitlement Usage
   * Reset the entitlement usage and start a new period. Eligible grants will be rolled over
   *
   * @example
   * const entitlement = await openmeter.subjects.resetEntitlementUsage('customer-1', 'ai_tokens', {
   *    retainAnchor: true
   * })
   */
  public async resetEntitlementUsage(
    subjectIdOrKey: string,
    entitlementIdOrFeatureIdOrFeatureKey: string,
    input: EntitlementResetInputs,
    options?: RequestOptions
  ): Promise<Entitlement> {
    return await this.request({
      path: `/api/v1/subjects/${subjectIdOrKey}/entitlements/${entitlementIdOrFeatureIdOrFeatureKey}/reset`,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(input),
      options,
    })
  }
}
