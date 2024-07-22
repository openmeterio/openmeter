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
import {
  EntitlementGrant,
  EntitlementGrantCreateInput,
  ListEntitlementGrantQueryParams,
} from './grant.js'

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
   * @remarks
   * Input should be either `EntitlementMeteredCreateInputs`, `EntitlementStaticCreateInputs`, or `EntitlementBooleanCreateInputs`
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
   * List entitlements of a subject
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
   * Get entitlement
   *
   * @example
   * const entitlement = await openmeter.subjects.getEntitlement('customer-1', '01J1SD3QDV86GP77TQ4PZZ4EXE')
   */
  public async getEntitlement(
    subjectIdOrKey: string,
    entitlementId: string,
    options?: RequestOptions
  ): Promise<Entitlement> {
    return await this.request({
      path: `/api/v1/subjects/${subjectIdOrKey}/entitlements/${entitlementId}`,
      method: 'GET',
      options,
    })
  }

  /**
   * Delete entitlement
   * @example
   * await openmeter.deleteEntitlement('customer-1', '01J1SD3QDV86GP77TQ4PZZ4EXE')
   */
  public async deleteEntitlement(
    subjectIdOrKey: string,
    entitlementId: string,
    options?: RequestOptions
  ): Promise<void> {
    return await this.request({
      path: `/api/v1/subjects/${subjectIdOrKey}/entitlements/${entitlementId}`,
      method: 'DELETE',
      options,
    })
  }

  /**
   * Get entitlement value by ID or Feature Key
   *
   * @example
   * const value = await openmeter.subjects.getEntitlementValue('customer-1', 'ai_tokens')
   */
  public async getEntitlementValue(
    subjectIdOrKey: string,
    entitlementIdOrFeatureKey: string,
    options?: RequestOptions
  ): Promise<EntitlementValue> {
    return await this.request({
      path: `/api/v1/subjects/${subjectIdOrKey}/entitlements/${entitlementIdOrFeatureKey}/value`,
      method: 'GET',
      options,
    })
  }

  /**
   * Get entitlement value at a specific time.
   *
   * @example
   * const value = await openmeter.subjects.getEntitlementValueAt('customer-1', 'ai_tokens', new Date('2024-01-01'))
   */
  public async getEntitlementValueAt(
    subjectIdOrKey: string,
    entitlementIdOrFeatureKey: string,
    at: Date,
    options?: RequestOptions
  ): Promise<EntitlementValue> {
    const searchParams = BaseClient.toURLSearchParams({ time: at })
    return await this.request({
      path: `/api/v1/subjects/${subjectIdOrKey}/entitlements/${entitlementIdOrFeatureKey}/value`,
      method: 'GET',
      options,
      searchParams,
    })
  }

  /**
   * Get entitlement history
   * @example
   * const entitlement = await openmeter.subjects.getEntitlementHistory('customer-1', '01J1SD3QDV86GP77TQ4PZZ4EXE')
   */
  public async getEntitlementHistory(
    subjectIdOrKey: string,
    entitlementId: string,
    params?: GetEntitlementHistoryQueryParams,
    options?: RequestOptions
  ): Promise<WindowedBalanceHistory[]> {
    const searchParams = params
      ? BaseClient.toURLSearchParams(params)
      : undefined
    return await this.request({
      path: `/api/v1/subjects/${subjectIdOrKey}/entitlements/${entitlementId}/history`,
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
   * const entitlement = await openmeter.subjects.resetEntitlementUsage('customer-1', '01J1SD3QDV86GP77TQ4PZZ4EXE', {
   *    retainAnchor: true
   * })
   */
  public async resetEntitlementUsage(
    subjectIdOrKey: string,
    entitlementId: string,
    input: EntitlementResetInputs,
    options?: RequestOptions
  ): Promise<Entitlement> {
    return await this.request({
      path: `/api/v1/subjects/${subjectIdOrKey}/entitlements/${entitlementId}/reset`,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(input),
      options,
    })
  }

  /** Entitlement Grant **/

  /**
   * Create Entitlement Grant
   * Create a grant for an entitlement.
   *
   * @example
   * const grant = await openmeter.subjects.createEntitlementGrant('customer-1', '01J1SD3QDV86GP77TQ4PZZ4EXE', {
   *    amount: 100,
   *    priority: 1,
   *    effectiveAt: '2023-01-01T00:00:00Z',
   *    expiration: {
   *      duration: 'HOUR',
   *      count: 12,
   *    },
   *    minRolloverAmount: 100,
   *    maxRolloverAmount: 100,
   *    recurrence: {
   *      interval: 'MONTH',
   *      anchor: '2024-06-28T18:29:44.867Z',
   *    },
   * })
   */
  public async createEntitlementGrant(
    subjectIdOrKey: string,
    entitlementIdOrFeatureKey: string,
    input: EntitlementGrantCreateInput,
    options?: RequestOptions
  ): Promise<EntitlementGrant> {
    return await this.request({
      path: `/api/v1/subjects/${subjectIdOrKey}/entitlements/${entitlementIdOrFeatureKey}/grants`,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(input),
      options,
    })
  }

  /**
   * List entitlement grants
   * @example
   * const entitlement = await openmeter.subjects.listEntitlementGrants('customer-1', '01J1SD3QDV86GP77TQ4PZZ4EXE')
   */
  public async listEntitlementGrants(
    subjectIdOrKey: string,
    entitlementIdOrFeatureKey: string,
    params?: ListEntitlementGrantQueryParams,
    options?: RequestOptions
  ): Promise<EntitlementGrant[]> {
    const searchParams = params
      ? BaseClient.toURLSearchParams(params)
      : undefined
    return await this.request({
      path: `/api/v1/subjects/${subjectIdOrKey}/entitlements/${entitlementIdOrFeatureKey}/grants`,
      method: 'GET',
      searchParams,
      options,
    })
  }
}
