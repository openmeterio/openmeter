import type { Client } from 'openapi-fetch'
import type { RequestOptions } from './common.js'
import type {
  CreateStripeCustomerPortalSessionParams,
  CustomerAppData,
  CustomerCreate,
  CustomerReplaceUpdate,
  operations,
  paths,
  StripeCustomerAppDataBase,
} from './schemas.js'
import { transformResponse } from './utils.js'

/**
 * Customers
 * Manage customer subscription lifecycles and plan assignments.
 */
export class Customers {
  public apps: CustomerApps
  public entitlementsV1: CustomerEntitlements
  public entitlements: CustomerEntitlementsV2
  public stripe: CustomerStripe

  constructor(private client: Client<paths, `${string}/${string}`>) {
    this.apps = new CustomerApps(client)
    this.entitlementsV1 = new CustomerEntitlements(client)
    this.entitlements = new CustomerEntitlementsV2(client)
    this.stripe = new CustomerStripe(client)
  }

  /**
   * Create a customer
   * @param customer - The customer to create
   * @param signal - An optional abort signal
   * @returns The created customer
   */
  public async create(customer: CustomerCreate, options?: RequestOptions) {
    const resp = await this.client.POST('/api/v1/customers', {
      body: customer,
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get a customer by ID
   * @param customerIdOrKey - The ID or Key of the customer
   * @param signal - An optional abort signal
   * @returns The customer
   */
  public async get(
    customerIdOrKey: operations['getCustomer']['parameters']['path']['customerIdOrKey'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/customers/{customerIdOrKey}', {
      params: {
        path: {
          customerIdOrKey,
        },
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Update a customer
   * @param customerIdOrKey - The ID or Key of the customer
   * @param customer - The customer to update
   * @param signal - An optional abort signal
   * @returns The updated customer
   */
  public async update(
    customerIdOrKey: operations['updateCustomer']['parameters']['path']['customerIdOrKey'],
    customer: CustomerReplaceUpdate,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT('/api/v1/customers/{customerIdOrKey}', {
      body: customer,
      params: {
        path: {
          customerIdOrKey,
        },
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Delete a customer
   * @param customerIdOrKey - The ID or Key of the customer
   * @param signal - An optional abort signal
   * @returns The deleted customer
   */
  public async delete(
    customerIdOrKey: operations['deleteCustomer']['parameters']['path']['customerIdOrKey'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.DELETE(
      '/api/v1/customers/{customerIdOrKey}',
      {
        params: {
          path: {
            customerIdOrKey,
          },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * List customers
   * @param signal - An optional abort signal
   * @returns The list of customers
   */
  public async list(
    query?: operations['listCustomers']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/customers', {
      params: {
        query,
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get customer access
   * @param customerIdOrKey - The ID or Key of the customer
   * @param options - Optional request options
   * @returns The customer access information
   */
  public async getAccess(
    customerIdOrKey: operations['getCustomerAccess']['parameters']['path']['customerIdOrKey'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/api/v1/customers/{customerIdOrKey}/access',
      {
        params: {
          path: {
            customerIdOrKey,
          },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * List customer subscriptions
   * @param customerIdOrKey - The ID or key of the customer
   * @param query - The query parameters
   * @param signal - An optional abort signal
   * @returns The list of customer subscriptions
   */
  public async listSubscriptions(
    customerIdOrKey: operations['listCustomerSubscriptions']['parameters']['path']['customerIdOrKey'],
    query?: operations['listCustomerSubscriptions']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/api/v1/customers/{customerIdOrKey}/subscriptions',
      {
        params: { path: { customerIdOrKey }, query },
        ...options,
      },
    )

    return transformResponse(resp)
  }
}

/**
 * Customer Apps
 * Manage customer apps.
 */
export class CustomerApps {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Upsert customer app data
   * @param customerIdOrKey - The ID or Key of the customer
   * @param appData - The app data to upsert
   * @param signal - An optional abort signal
   * @returns The upserted app data
   */
  public async upsert(
    customerIdOrKey: operations['upsertCustomerAppData']['parameters']['path']['customerIdOrKey'],
    appData: CustomerAppData[],
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT(
      '/api/v1/customers/{customerIdOrKey}/apps',
      {
        body: appData,
        params: {
          path: {
            customerIdOrKey,
          },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * List customer app data
   * @param customerIdOrKey - The ID or key of the customer
   * @param query - The query parameters
   * @param signal - An optional abort signal
   * @returns The list of customer app data
   */
  public async list(
    customerIdOrKey: operations['listCustomerAppData']['parameters']['path']['customerIdOrKey'],
    query?: operations['listCustomerAppData']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/api/v1/customers/{customerIdOrKey}/apps',
      {
        params: {
          path: { customerIdOrKey },
          query,
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Delete customer app data
   * @param customerIdOrKey - The ID or key of the customer
   * @param appId - The ID of the app
   * @param signal - An optional abort signal
   * @returns The deleted customer app data
   */
  public async delete(
    customerIdOrKey: operations['deleteCustomerAppData']['parameters']['path']['customerIdOrKey'],
    appId: operations['deleteCustomerAppData']['parameters']['path']['appId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.DELETE(
      '/api/v1/customers/{customerIdOrKey}/apps/{appId}',
      {
        params: { path: { appId, customerIdOrKey } },
        ...options,
      },
    )

    return transformResponse(resp)
  }
}

/**
 * Customer Stripe
 * Manage customer Stripe data.
 */
export class CustomerStripe {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Upsert customer stripe app data
   * @param customerIdOrKey - The ID or Key of the customer
   * @param appData - The app data to upsert
   * @param signal - An optional abort signal
   * @returns The upserted customer stripe app data
   */
  public async upsert(
    customerIdOrKey: operations['upsertCustomerStripeAppData']['parameters']['path']['customerIdOrKey'],
    stripeAppData: StripeCustomerAppDataBase,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT(
      '/api/v1/customers/{customerIdOrKey}/stripe',
      {
        body: stripeAppData,
        params: {
          path: {
            customerIdOrKey,
          },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Get customer stripe app data
   * @param customerIdOrKey - The ID or key of the customer
   * @param query - The query parameters
   * @param signal - An optional abort signal
   * @returns The customer stripe app data
   */
  public async get(
    customerIdOrKey: operations['getCustomerStripeAppData']['parameters']['path']['customerIdOrKey'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/api/v1/customers/{customerIdOrKey}/stripe',
      {
        params: {
          path: { customerIdOrKey },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Create a Stripe customer portal session
   * @param customerIdOrKey - The ID or Key of the customer
   * @param params - The parameters for creating a Stripe customer portal session
   * @param signal - An optional abort signal
   * @returns The Stripe customer portal session
   */
  public async createPortalSession(
    customerIdOrKey: operations['createCustomerStripePortalSession']['parameters']['path']['customerIdOrKey'],
    params: CreateStripeCustomerPortalSessionParams,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/api/v1/customers/{customerIdOrKey}/stripe/portal',
      {
        body: params,
        params: {
          path: {
            customerIdOrKey,
          },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }
}

/**
 * Customer Entitlements
 */
export class CustomerEntitlements {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Get the value of an entitlement for a customer
   * @param customerIdOrKey - The ID or Key of the customer
   * @param featureKey - The key of the feature
   * @param signal - An optional abort signal
   * @returns The value of the entitlement
   */
  public async value(
    customerIdOrKey: operations['getCustomerEntitlementValue']['parameters']['path']['customerIdOrKey'],
    featureKey: operations['getCustomerEntitlementValue']['parameters']['path']['featureKey'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/api/v1/customers/{customerIdOrKey}/entitlements/{featureKey}/value',
      {
        params: { path: { customerIdOrKey, featureKey } },
        ...options,
      },
    )

    return transformResponse(resp)
  }
}

/**
 * Customer Entitlements V2
 */
export class CustomerEntitlementsV2 {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * List all entitlements for a customer
   * @param customerIdOrKey - The ID or Key of the customer
   * @param options - Request options including query parameters
   * @returns List of customer entitlements
   */
  public async list(
    customerIdOrKey: operations['listCustomerEntitlementsV2']['parameters']['path']['customerIdOrKey'],
    options?: RequestOptions & {
      query?: operations['listCustomerEntitlementsV2']['parameters']['query']
    },
  ) {
    const resp = await this.client.GET(
      '/api/v2/customers/{customerIdOrKey}/entitlements',
      {
        params: {
          path: { customerIdOrKey },
          query: options?.query,
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Create a customer entitlement
   * @param customerIdOrKey - The ID or Key of the customer
   * @param entitlement - The entitlement data to create
   * @param options - Request options
   * @returns The created entitlement
   */
  public async create(
    customerIdOrKey: operations['createCustomerEntitlementV2']['parameters']['path']['customerIdOrKey'],
    entitlement: operations['createCustomerEntitlementV2']['requestBody']['content']['application/json'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/api/v2/customers/{customerIdOrKey}/entitlements',
      {
        body: entitlement,
        params: {
          path: { customerIdOrKey },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Get a specific customer entitlement
   * @param customerIdOrKey - The ID or Key of the customer
   * @param entitlementIdOrFeatureKey - The ID or feature key of the entitlement
   * @param options - Request options
   * @returns The entitlement
   */
  public async get(
    customerIdOrKey: operations['getCustomerEntitlementV2']['parameters']['path']['customerIdOrKey'],
    entitlementIdOrFeatureKey: operations['getCustomerEntitlementV2']['parameters']['path']['entitlementIdOrFeatureKey'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}',
      {
        params: {
          path: { customerIdOrKey, entitlementIdOrFeatureKey },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Delete a customer entitlement
   * @param customerIdOrKey - The ID or Key of the customer
   * @param entitlementIdOrFeatureKey - The ID or feature key of the entitlement
   * @param options - Request options
   * @returns The deletion response
   */
  public async delete(
    customerIdOrKey: operations['deleteCustomerEntitlementV2']['parameters']['path']['customerIdOrKey'],
    entitlementIdOrFeatureKey: operations['deleteCustomerEntitlementV2']['parameters']['path']['entitlementIdOrFeatureKey'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.DELETE(
      '/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}',
      {
        params: {
          path: { customerIdOrKey, entitlementIdOrFeatureKey },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Override a customer entitlement
   * @param customerIdOrKey - The ID or Key of the customer
   * @param entitlementIdOrFeatureKey - The ID or feature key of the entitlement
   * @param entitlement - The new entitlement data
   * @param options - Request options
   * @returns The overridden entitlement
   */
  public async override(
    customerIdOrKey: operations['overrideCustomerEntitlementV2']['parameters']['path']['customerIdOrKey'],
    entitlementIdOrFeatureKey: operations['overrideCustomerEntitlementV2']['parameters']['path']['entitlementIdOrFeatureKey'],
    entitlement: operations['overrideCustomerEntitlementV2']['requestBody']['content']['application/json'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT(
      '/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/override',
      {
        body: entitlement,
        params: {
          path: { customerIdOrKey, entitlementIdOrFeatureKey },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * List grants for a customer entitlement
   * @param customerIdOrKey - The ID or Key of the customer
   * @param entitlementIdOrFeatureKey - The ID or feature key of the entitlement
   * @param options - Request options including query parameters
   * @returns List of entitlement grants
   */
  public async listGrants(
    customerIdOrKey: operations['listCustomerEntitlementGrantsV2']['parameters']['path']['customerIdOrKey'],
    entitlementIdOrFeatureKey: operations['listCustomerEntitlementGrantsV2']['parameters']['path']['entitlementIdOrFeatureKey'],
    options?: RequestOptions & {
      query?: operations['listCustomerEntitlementGrantsV2']['parameters']['query']
    },
  ) {
    const resp = await this.client.GET(
      '/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants',
      {
        params: {
          path: { customerIdOrKey, entitlementIdOrFeatureKey },
          query: options?.query,
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Create a grant for a customer entitlement
   * @param customerIdOrKey - The ID or Key of the customer
   * @param entitlementIdOrFeatureKey - The ID or feature key of the entitlement
   * @param grant - The grant data to create
   * @param options - Request options
   * @returns The created grant
   */
  public async createGrant(
    customerIdOrKey: operations['createCustomerEntitlementGrantV2']['parameters']['path']['customerIdOrKey'],
    entitlementIdOrFeatureKey: operations['createCustomerEntitlementGrantV2']['parameters']['path']['entitlementIdOrFeatureKey'],
    grant: operations['createCustomerEntitlementGrantV2']['requestBody']['content']['application/json'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants',
      {
        body: grant,
        params: {
          path: { customerIdOrKey, entitlementIdOrFeatureKey },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Get the value of a customer entitlement
   * @param customerIdOrKey - The ID or Key of the customer
   * @param entitlementIdOrFeatureKey - The ID or feature key of the entitlement
   * @param options - Request options including query parameters
   * @returns The entitlement value
   */
  public async value(
    customerIdOrKey: operations['getCustomerEntitlementValueV2']['parameters']['path']['customerIdOrKey'],
    entitlementIdOrFeatureKey: operations['getCustomerEntitlementValueV2']['parameters']['path']['entitlementIdOrFeatureKey'],
    options?: RequestOptions & {
      query?: operations['getCustomerEntitlementValueV2']['parameters']['query']
    },
  ) {
    const resp = await this.client.GET(
      '/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/value',
      {
        params: {
          path: { customerIdOrKey, entitlementIdOrFeatureKey },
          query: options?.query,
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Get the history of a customer entitlement
   * @param customerIdOrKey - The ID or Key of the customer
   * @param entitlementIdOrFeatureKey - The ID or feature key of the entitlement
   * @param windowSize - The window size for the history
   * @param options - Request options including query parameters
   * @returns The entitlement history
   */
  public async history(
    customerIdOrKey: operations['getCustomerEntitlementHistoryV2']['parameters']['path']['customerIdOrKey'],
    entitlementIdOrFeatureKey: operations['getCustomerEntitlementHistoryV2']['parameters']['path']['entitlementIdOrFeatureKey'],
    windowSize: operations['getCustomerEntitlementHistoryV2']['parameters']['query']['windowSize'],
    options?: RequestOptions & {
      query?: Omit<
        operations['getCustomerEntitlementHistoryV2']['parameters']['query'],
        'windowSize'
      >
    },
  ) {
    const resp = await this.client.GET(
      '/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/history',
      {
        params: {
          path: { customerIdOrKey, entitlementIdOrFeatureKey },
          query: {
            windowSize,
            ...options?.query,
          },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Reset the usage of a customer entitlement
   * @param customerIdOrKey - The ID or Key of the customer
   * @param entitlementIdOrFeatureKey - The ID or feature key of the entitlement
   * @param reset - The reset data
   * @param options - Request options
   * @returns The reset response
   */
  public async resetUsage(
    customerIdOrKey: operations['resetCustomerEntitlementUsageV2']['parameters']['path']['customerIdOrKey'],
    entitlementIdOrFeatureKey: operations['resetCustomerEntitlementUsageV2']['parameters']['path']['entitlementIdOrFeatureKey'],
    reset: operations['resetCustomerEntitlementUsageV2']['requestBody']['content']['application/json'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/reset',
      {
        body: reset,
        params: {
          path: { customerIdOrKey, entitlementIdOrFeatureKey },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }
}
