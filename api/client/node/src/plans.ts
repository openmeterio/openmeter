import { transformResponse } from './utils.js'
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
  public async create(plan: PlanCreate, signal?: AbortSignal) {
    const resp = await this.client.POST('/api/v1/plans', {
      signal,
      body: plan,
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
    signal?: AbortSignal
  ) {
    const resp = await this.client.GET('/api/v1/plans/{planId}', {
      signal,
      params: {
        path: {
          planId: id,
        },
      },
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
    signal?: AbortSignal
  ) {
    const resp = await this.client.GET('/api/v1/plans', {
      signal,
      params: {
        query,
      },
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
    signal?: AbortSignal
  ) {
    const resp = await this.client.PUT('/api/v1/plans/{planId}', {
      signal,
      body: plan,
      params: {
        path: {
          planId: id,
        },
      },
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
    signal?: AbortSignal
  ) {
    const resp = await this.client.DELETE('/api/v1/plans/{planId}', {
      signal,
      params: {
        path: {
          planId: id,
        },
      },
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
    signal?: AbortSignal
  ) {
    const resp = await this.client.POST('/api/v1/plans/{planId}/archive', {
      signal,
      params: {
        path: {
          planId: id,
        },
      },
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
    signal?: AbortSignal
  ) {
    const resp = await this.client.POST('/api/v1/plans/{planId}/publish', {
      signal,
      params: {
        path: {
          planId: id,
        },
      },
    })

    return transformResponse(resp)
  }
}
