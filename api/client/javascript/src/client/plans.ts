import { transformResponse } from './utils.js'
import type { RequestOptions } from './common.js'
import type {
  operations,
  paths,
  PlanCreate,
  PlanReplaceUpdate,
} from './schemas.js'
import type { Client } from 'openapi-fetch'

export class Plans {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Create a plan
   * @param plan - The plan to create
   * @param signal - An optional abort signal
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
   * Get a plan by ID or key
   * @param idOrKey - The ID or key of the plan
   * @param signal - An optional abort signal
   * @returns The plan
   */
  public async get(
    id: operations['getPlan']['parameters']['path']['planId'],
    options?: RequestOptions
  ) {
    const resp = await this.client.GET('/api/v1/plans/{planId}', {
      params: {
        path: {
          planId: id,
        },
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List plans
   * @param query - The query parameters
   * @param signal - An optional abort signal
   * @returns The list of plans
   */
  public async list(
    query?: operations['listPlans']['parameters']['query'],
    options?: RequestOptions
  ) {
    const resp = await this.client.GET('/api/v1/plans', {
      params: {
        query,
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Update a plan
   * @param id - The ID of the plan
   * @param plan - The plan to update
   * @param signal - An optional abort signal
   * @returns The updated plan
   */
  public async update(
    id: operations['updatePlan']['parameters']['path']['planId'],
    plan: PlanReplaceUpdate,
    options?: RequestOptions
  ) {
    const resp = await this.client.PUT('/api/v1/plans/{planId}', {
      body: plan,
      params: {
        path: {
          planId: id,
        },
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Delete a plan
   * @param id - The ID of the plan
   * @param signal - An optional abort signal
   * @returns The deleted plan
   */
  public async delete(
    id: operations['deletePlan']['parameters']['path']['planId'],
    options?: RequestOptions
  ) {
    const resp = await this.client.DELETE('/api/v1/plans/{planId}', {
      params: {
        path: {
          planId: id,
        },
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Archive a plan
   * @param id - The ID of the plan
   * @param signal - An optional abort signal
   * @returns The archived plan
   */
  public async archive(
    id: operations['archivePlan']['parameters']['path']['planId'],
    options?: RequestOptions
  ) {
    const resp = await this.client.POST('/api/v1/plans/{planId}/archive', {
      params: {
        path: {
          planId: id,
        },
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Publish a plan
   * @param id - The ID of the plan
   * @param signal - An optional abort signal
   * @returns The published plan
   */
  public async publish(
    id: operations['publishPlan']['parameters']['path']['planId'],
    options?: RequestOptions
  ) {
    const resp = await this.client.POST('/api/v1/plans/{planId}/publish', {
      params: {
        path: {
          planId: id,
        },
      },
      ...options,
    })

    return transformResponse(resp)
  }
}
