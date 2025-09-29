import type { Client } from 'openapi-fetch'
import type { RequestOptions } from './common.js'
import type {
  Entitlement,
  EntitlementCreateInputs,
  EntitlementGrantCreateInput,
  operations,
  paths,
  ResetEntitlementUsageInput,
} from './schemas.js'
import { transformResponse } from './utils.js'

/**
 * Entitlements
 * @description With Entitlements, you can define and enforce usage limits, implement quota-based pricing, and manage access to features in your application.
 */
export class Entitlements {
  public grants: Grants

  constructor(private client: Client<paths, `${string}/${string}`>) {
    this.grants = new Grants(client)
  }

  /**
   * Create an entitlement
   *
   *     - Boolean entitlements define static feature access, e.g. "Can use SSO authentication".
   *     - Static entitlements let you pass along a configuration while granting access, e.g. "Using this feature with X Y settings" (passed in the config).
   *     - Metered entitlements have many use cases, from setting up usage-based access to implementing complex credit systems.  Example: The customer can use 10000 AI tokens during the usage period of the entitlement.
   *
   *     A given subject can only have one active (non-deleted) entitlement per featureKey. If you try to create a new entitlement for a featureKey that already has an active entitlement, the request will fail with a 409 error.
   *
   *     Once an entitlement is created you cannot modify it, only delete it.
   *
   * @param subjectIdOrKey - The ID or key of the subject
   * @param entitlement - The entitlement to create
   * @param signal - An optional abort signal
   * @returns The created entitlement
   */
  public async create(
    subjectIdOrKey: operations['createEntitlement']['parameters']['path']['subjectIdOrKey'],
    entitlement: EntitlementCreateInputs,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/api/v1/subjects/{subjectIdOrKey}/entitlements',
      {
        body: entitlement,
        params: {
          path: {
            subjectIdOrKey: subjectIdOrKey,
          },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Get an entitlement by ID
   *
   * @param id - The ID of the entitlement
   * @param signal - An optional abort signal
   * @returns The entitlement
   */
  public async get(
    id: operations['getEntitlement']['parameters']['path']['entitlementId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/entitlements/{entitlementId}', {
      params: {
        path: {
          entitlementId: id,
        },
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List entitlements
   *
   * @param query - The query parameters
   * @param signal - An optional abort signal
   * @returns The entitlements
   */
  public async list(
    query?: Omit<
      operations['listEntitlements']['parameters']['query'],
      'page' | 'pageSize'
    >,
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/entitlements', {
      params: {
        query,
      },
      ...options,
    })

    return transformResponse(resp) as Entitlement[]
  }

  /**
   * Delete an entitlement
   *
   * @param subjectIdOrKey - The ID or key of the subject
   * @param entitlementId - The ID of the entitlement
   * @param signal - An optional abort signal
   * @returns The deleted entitlement
   */
  public async delete(
    subjectIdOrKey: operations['deleteEntitlement']['parameters']['path']['subjectIdOrKey'],
    entitlementId: operations['deleteEntitlement']['parameters']['path']['entitlementId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.DELETE(
      '/api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId}',
      {
        params: {
          path: {
            entitlementId,
            subjectIdOrKey,
          },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Get the value of an entitlement to check access
   * All entitlement types share the hasAccess property in their value response, but multiple other properties are returned based on the entitlement type.
   *
   * @param subjectIdOrKey - The ID or key of the subject
   * @param entitlementIdOrFeatureKey - The ID or feature key of the entitlement
   * @param query - The query parameters
   * @param signal - An optional abort signal
   * @returns The entitlement value
   */
  public async value(
    subjectIdOrKey: operations['getEntitlementValue']['parameters']['path']['subjectIdOrKey'],
    entitlementIdOrFeatureKey: operations['getEntitlementValue']['parameters']['path']['entitlementIdOrFeatureKey'],
    query?: operations['getEntitlementValue']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/value',
      {
        params: {
          path: {
            entitlementIdOrFeatureKey,
            subjectIdOrKey,
          },
          query,
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Get the history of an entitlement
   * Returns historical balance and usage data for the entitlement. The queried history can span accross multiple reset events.
   *
   * @param subjectIdOrKey - The ID or key of the subject
   * @param entitlementId - The ID of the entitlement
   * @param query - The query parameters
   * @param signal - An optional abort signal
   * @returns The history of the entitlement
   */
  public async history(
    subjectIdOrKey: operations['getEntitlementHistory']['parameters']['path']['subjectIdOrKey'],
    entitlementId: operations['getEntitlementHistory']['parameters']['path']['entitlementId'],
    query: operations['getEntitlementHistory']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId}/history',
      {
        params: {
          path: {
            entitlementId,
            subjectIdOrKey,
          },
          query,
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Override an entitlement
   * This is useful for upgrades, downgrades, or other changes to entitlements that require a new entitlement to be created with zero downtime.
   *
   * @param subjectIdOrKey - The ID or key of the subject
   * @param entitlementIdOrFeatureKey - The ID or feature key of the entitlement
   * @param override - The override to create
   * @param signal - An optional abort signal
   * @returns The overridden entitlement
   */
  public async override(
    subjectIdOrKey: operations['overrideEntitlement']['parameters']['path']['subjectIdOrKey'],
    entitlementIdOrFeatureKey: operations['overrideEntitlement']['parameters']['path']['entitlementIdOrFeatureKey'],
    override: EntitlementCreateInputs,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT(
      '/api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/override',
      {
        body: override,
        params: {
          path: {
            entitlementIdOrFeatureKey,
            subjectIdOrKey,
          },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Reset entitlement usage
   * - Reset marks the start of a new usage period for the entitlement and initiates grant rollover. At the start of a period usage is zerod out and grants are rolled over based on their rollover settings. It would typically be synced with the subjects billing period to enforce usage based on their subscription.
   * - Usage is automatically reset for metered entitlements based on their usage period, but this endpoint allows to manually reset it at any time. When doing so the period anchor of the entitlement can be changed if needed.
   *
   * @param subjectIdOrKey - The ID or key of the subject
   * @param entitlementId - The ID of the entitlement
   * @param body - The body of the request
   * @param signal - An optional abort signal
   * @returns The reset entitlement
   */
  public async reset(
    subjectIdOrKey: operations['resetEntitlementUsage']['parameters']['path']['subjectIdOrKey'],
    entitlementId: operations['resetEntitlementUsage']['parameters']['path']['entitlementId'],
    body: ResetEntitlementUsageInput,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId}/reset',
      {
        body,
        params: {
          path: {
            entitlementId,
            subjectIdOrKey,
          },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }
}

export class Grants {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Grant usage to a subject for an entitlement
   *
   * @param subjectIdOrKey - The ID or key of the subject
   * @param entitlementIdOrFeatureKey - The ID or feature key of the entitlement
   * @param grant - The grant to create
   * @param signal - An optional abort signal
   * @returns The created grant
   */
  public async create(
    subjectIdOrKey: operations['createGrant']['parameters']['path']['subjectIdOrKey'],
    entitlementIdOrFeatureKey: operations['createGrant']['parameters']['path']['entitlementIdOrFeatureKey'],
    grant: EntitlementGrantCreateInput,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants',
      {
        body: grant,
        params: {
          path: {
            entitlementIdOrFeatureKey,
            subjectIdOrKey,
          },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * List grants for an entitlement
   *
   * @param subjectIdOrKey - The ID or key of the subject
   * @param entitlementIdOrFeatureKey - The ID or feature key of the entitlement
   * @param signal - An optional abort signal
   * @returns The grants
   */
  public async list(
    subjectIdOrKey: operations['listEntitlementGrants']['parameters']['path']['subjectIdOrKey'],
    entitlementIdOrFeatureKey: operations['listEntitlementGrants']['parameters']['path']['entitlementIdOrFeatureKey'],
    query?: operations['listEntitlementGrants']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants',
      {
        params: {
          path: {
            entitlementIdOrFeatureKey,
            subjectIdOrKey,
          },
          query,
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * List all grants
   * List all grants for all the subjects and entitlements.
   *
   * @param query - The query parameters
   * @param options - The request options
   * @returns The grants
   */
  public async listAll(
    query?: operations['listGrants']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/grants', {
      params: {
        query,
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Void a grant
   * Voiding a grant means it is no longer valid, it doesn't take part in further balance calculations.
   * Voiding a grant does not retroactively take effect, meaning any usage that has already been attributed
   * to the grant will remain, but future usage cannot be burnt down from the grant.
   *
   * @param grantId - The ID of the grant
   * @param options - The request options
   * @returns The voided grant
   */
  public async void(
    grantId: operations['voidGrant']['parameters']['path']['grantId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.DELETE('/api/v1/grants/{grantId}', {
      params: {
        path: {
          grantId,
        },
      },
      ...options,
    })

    return transformResponse(resp)
  }
}

/**
 * Entitlements V2
 * @description With Entitlements, you can define and enforce usage limits, implement quota-based pricing, and manage access to features in your application.
 */
export class EntitlementsV2 {
  public grants: GrantsV2

  constructor(private client: Client<paths, `${string}/${string}`>) {
    this.grants = new GrantsV2(client)
  }

  /**
   * List all entitlements for all customers and features
   * @description This endpoint is intended for administrative purposes only.
   * To fetch entitlements of a specific customer, use the customer entitlements endpoint.
   * @param options - Request options including query parameters
   * @returns List of entitlements
   */
  public async list(
    options?: RequestOptions & {
      query?: operations['listEntitlementsV2']['parameters']['query']
    },
  ) {
    const resp = await this.client.GET('/api/v2/entitlements', {
      params: {
        query: options?.query,
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get entitlement by ID
   * @param entitlementId - The ID of the entitlement
   * @param options - Request options
   * @returns The entitlement
   */
  public async get(
    entitlementId: operations['getEntitlementByIdV2']['parameters']['path']['entitlementId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v2/entitlements/{entitlementId}', {
      params: {
        path: { entitlementId },
      },
      ...options,
    })

    return transformResponse(resp)
  }
}

/**
 * Grants
 */
export class GrantsV2 {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * List all grants for all customers and entitlements
   * @description This endpoint is intended for administrative purposes only.
   * To fetch grants of a specific entitlement, use the customer entitlements grants endpoint.
   * @param options - Request options including query parameters
   * @returns List of grants
   */
  public async list(
    options?: RequestOptions & {
      query?: operations['listGrantsV2']['parameters']['query']
    },
  ) {
    const resp = await this.client.GET('/api/v2/grants', {
      params: {
        query: options?.query,
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Void a grant (legacy method using V1 endpoint)
   * @description Voiding a grant means it is no longer valid, it doesn't take part in further balance calculations.
   * Voiding a grant does not retroactively take effect, meaning any usage that has already been attributed
   * to the grant will remain, but future usage cannot be burnt down from the grant.
   * @param grantId - The ID of the grant
   * @param options - Request options
   * @returns The voided grant
   * @deprecated This method uses the legacy V1 endpoint. Consider using customer-specific grant operations instead.
   */
  public async void(
    grantId: operations['voidGrant']['parameters']['path']['grantId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.DELETE('/api/v1/grants/{grantId}', {
      params: {
        path: {
          grantId,
        },
      },
      ...options,
    })

    return transformResponse(resp)
  }
}
