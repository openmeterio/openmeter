import type { Client } from 'openapi-fetch'
import type { RequestOptions } from './common.js'
import type {
  operations,
  PlanCreate,
  PlanReplaceUpdate,
  paths,
} from './schemas.js'
import { transformResponse } from './utils.js'

/**
 * Plans
 * Manage customer subscription plans and addon assignments.
 */
export class Plans {
  public addons: PlanAddons

  constructor(private client: Client<paths, `${string}/${string}`>) {
    this.addons = new PlanAddons(this.client)
  }

  /**
   * Create a plan
   * @param plan - The plan to create
   * @param options - Optional request options
   * @returns The created plan
   */
  public async create(plan: PlanCreate, options?: RequestOptions) {
    const resp = await this.client.POST('/api/v1/plans', {
      body: plan,
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get a plan by ID
   * @param planId - The ID of the plan to retrieve
   * @param params - Optional query parameters
   * @param options - Optional request options
   * @returns The plan
   */
  public async get(
    planId: operations['getPlan']['parameters']['path']['planId'],
    params?: operations['getPlan']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/plans/{planId}', {
      params: {
        path: { planId },
        query: params,
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List plans
   * @param params - Optional parameters for listing plans
   * @param options - Optional request options
   * @returns A list of plans
   */
  public async list(
    params?: operations['listPlans']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/plans', {
      params: { query: params },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Update a plan
   * @param planId - The ID of the plan to update
   * @param plan - The plan data to update
   * @param options - Optional request options
   * @returns The updated plan
   */
  public async update(
    planId: operations['updatePlan']['parameters']['path']['planId'],
    plan: PlanReplaceUpdate,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT('/api/v1/plans/{planId}', {
      body: plan,
      params: { path: { planId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Delete a plan by ID
   * @param planId - The ID of the plan to delete
   * @param options - Optional request options
   * @returns void or standard error response structure
   */
  public async delete(
    planId: operations['deletePlan']['parameters']['path']['planId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.DELETE('/api/v1/plans/{planId}', {
      params: { path: { planId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Archive a plan
   * @param planId - The ID of the plan to archive
   * @param options - Optional request options
   * @returns The archived plan
   */
  public async archive(
    planId: operations['archivePlan']['parameters']['path']['planId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST('/api/v1/plans/{planId}/archive', {
      params: { path: { planId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Publish a plan
   * @param planId - The ID of the plan to publish
   * @param options - Optional request options
   * @returns The published plan
   */
  public async publish(
    planId: operations['publishPlan']['parameters']['path']['planId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST('/api/v1/plans/{planId}/publish', {
      params: { path: { planId } },
      ...options,
    })

    return transformResponse(resp)
  }
}

/**
 * Plan Addons
 * Manage addon assignments for plans.
 */
export class PlanAddons {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * List plan addons
   * @param planId - The ID of the plan
   * @param params - Optional query parameters
   * @param options - Optional request options
   * @returns A list of plan addons
   */
  public async list(
    planId: operations['listPlanAddons']['parameters']['path']['planId'],
    params?: operations['listPlanAddons']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/plans/{planId}/addons', {
      params: {
        path: { planId },
        query: params,
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Create a plan addon
   * @param planId - The ID of the plan
   * @param planAddon - The plan addon to create
   * @param options - Optional request options
   * @returns The created plan addon
   */
  public async create(
    planId: operations['createPlanAddon']['parameters']['path']['planId'],
    planAddon: operations['createPlanAddon']['requestBody']['content']['application/json'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST('/api/v1/plans/{planId}/addons', {
      body: planAddon,
      params: { path: { planId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get a plan addon by ID
   * @param planId - The ID of the plan
   * @param planAddonId - The ID of the plan addon
   * @param options - Optional request options
   * @returns The plan addon
   */
  public async get(
    planId: operations['getPlanAddon']['parameters']['path']['planId'],
    planAddonId: operations['getPlanAddon']['parameters']['path']['planAddonId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/api/v1/plans/{planId}/addons/{planAddonId}',
      {
        params: {
          path: { planAddonId, planId },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Update a plan addon
   * @param planId - The ID of the plan
   * @param planAddonId - The ID of the plan addon to update
   * @param planAddon - The plan addon data to update
   * @param options - Optional request options
   * @returns The updated plan addon
   */
  public async update(
    planId: operations['updatePlanAddon']['parameters']['path']['planId'],
    planAddonId: operations['updatePlanAddon']['parameters']['path']['planAddonId'],
    planAddon: operations['updatePlanAddon']['requestBody']['content']['application/json'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT(
      '/api/v1/plans/{planId}/addons/{planAddonId}',
      {
        body: planAddon,
        params: { path: { planAddonId, planId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Delete a plan addon by ID
   * @param planId - The ID of the plan
   * @param planAddonId - The ID of the plan addon to delete
   * @param options - Optional request options
   * @returns void or standard error response structure
   */
  public async delete(
    planId: operations['deletePlanAddon']['parameters']['path']['planId'],
    planAddonId: operations['deletePlanAddon']['parameters']['path']['planAddonId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.DELETE(
      '/api/v1/plans/{planId}/addons/{planAddonId}',
      {
        params: { path: { planAddonId, planId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }
}
