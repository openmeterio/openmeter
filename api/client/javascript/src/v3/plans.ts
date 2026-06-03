import type { Client } from 'openapi-fetch'
import type { RequestOptions } from '../client/common.js'
import { transformResponse } from '../client/utils.js'
import type {
  CreatePlanAddonRequest,
  CreatePlanRequest,
  operations,
  paths,
  UpsertPlanAddonRequest,
  UpsertPlanRequest,
} from './schemas.js'

/**
 * Plans (v3)
 *
 * Thin wrapper over the v3 plans endpoints. Request/response bodies use the v3
 * wire shape verbatim (snake_case); no field renaming (Option A).
 */
export class Plans {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Create a plan
   */
  public async create(plan: CreatePlanRequest, options?: RequestOptions) {
    const resp = await this.client.POST('/openmeter/plans', {
      body: plan,
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List plans
   */
  public async list(
    params?: operations['list-plans']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/openmeter/plans', {
      params: { query: params },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get a plan by ID
   */
  public async get(planId: string, options?: RequestOptions) {
    const resp = await this.client.GET('/openmeter/plans/{planId}', {
      params: { path: { planId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Publish a plan
   */
  public async publish(planId: string, options?: RequestOptions) {
    const resp = await this.client.POST('/openmeter/plans/{planId}/publish', {
      params: { path: { planId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Update (replace) a plan
   */
  public async update(
    planId: string,
    plan: UpsertPlanRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT('/openmeter/plans/{planId}', {
      body: plan,
      params: { path: { planId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Delete a plan
   */
  public async delete(planId: string, options?: RequestOptions) {
    const resp = await this.client.DELETE('/openmeter/plans/{planId}', {
      params: { path: { planId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Archive a plan
   */
  public async archive(planId: string, options?: RequestOptions) {
    const resp = await this.client.POST('/openmeter/plans/{planId}/archive', {
      params: { path: { planId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List add-ons associated with a plan
   */
  public async listAddons(
    planId: string,
    params?: operations['list-plan-addons']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/openmeter/plans/{planId}/addons', {
      params: { path: { planId }, query: params },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Add an add-on to a plan
   */
  public async createAddon(
    planId: string,
    addon: CreatePlanAddonRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST('/openmeter/plans/{planId}/addons', {
      body: addon,
      params: { path: { planId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get an add-on association for a plan
   */
  public async getAddon(
    planId: string,
    planAddonId: string,
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/openmeter/plans/{planId}/addons/{planAddonId}',
      {
        params: { path: { planAddonId, planId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Update an add-on association for a plan
   */
  public async updateAddon(
    planId: string,
    planAddonId: string,
    addon: UpsertPlanAddonRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT(
      '/openmeter/plans/{planId}/addons/{planAddonId}',
      {
        body: addon,
        params: { path: { planAddonId, planId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Remove an add-on from a plan
   */
  public async deleteAddon(
    planId: string,
    planAddonId: string,
    options?: RequestOptions,
  ) {
    const resp = await this.client.DELETE(
      '/openmeter/plans/{planId}/addons/{planAddonId}',
      {
        params: { path: { planAddonId, planId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }
}
