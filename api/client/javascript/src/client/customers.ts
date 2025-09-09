import { transformResponse } from './utils.js'
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
import type { Client } from 'openapi-fetch'

/**
 * Customers
 * Manage customer subscription lifecycles and plan assignments.
 */
export class Customers {
  public apps: CustomerApps
  public entitlements: CustomerEntitlements
  public stripe: CustomerStripe

  constructor(private client: Client<paths, `${string}/${string}`>) {
    this.apps = new CustomerApps(client)
    this.entitlements = new CustomerEntitlements(client)
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
    options?: RequestOptions
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
    options?: RequestOptions
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
    options?: RequestOptions
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
      }
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
    options?: RequestOptions
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
    options?: RequestOptions
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
      }
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
    options?: RequestOptions
  ) {
    const resp = await this.client.GET(
      '/api/v1/customers/{customerIdOrKey}/subscriptions',
      {
        params: { path: { customerIdOrKey }, query },
        ...options,
      }
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
    options?: RequestOptions
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
      }
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
    options?: RequestOptions
  ) {
    const resp = await this.client.GET(
      '/api/v1/customers/{customerIdOrKey}/apps',
      {
        params: {
          path: { customerIdOrKey },
          query,
        },
        ...options,
      }
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
    options?: RequestOptions
  ) {
    const resp = await this.client.DELETE(
      '/api/v1/customers/{customerIdOrKey}/apps/{appId}',
      {
        params: { path: { appId, customerIdOrKey } },
        ...options,
      }
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
    options?: RequestOptions
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
      }
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
    options?: RequestOptions
  ) {
    const resp = await this.client.GET(
      '/api/v1/customers/{customerIdOrKey}/apps',
      {
        params: {
          path: { customerIdOrKey },
        },
        ...options,
      }
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
    options?: RequestOptions
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
      }
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
   * Get the value of an entitlement for a customer (legacy v1)
   * @deprecated Use value for the v2 API instead.
   * @param customerIdOrKey - The ID or Key of the customer
   * @param featureKey - The key of the feature
   * @param signal - An optional abort signal
   * @returns The value of the entitlement
   */
  public async valueV1(
    customerIdOrKey: operations['getCustomerEntitlementValue']['parameters']['path']['customerIdOrKey'],
    featureKey: operations['getCustomerEntitlementValue']['parameters']['path']['featureKey'],
    options?: RequestOptions
  ) {
    const resp = await this.client.GET(
      '/api/v1/customers/{customerIdOrKey}/entitlements/{featureKey}/value',
      {
        params: { path: { customerIdOrKey, featureKey } },
        ...options,
      }
    )

    return transformResponse(resp)
  }

  /**
   * List customer entitlements
   * @param customerIdOrKey - The ID or Key of the customer
   * @param query - The query parameters
   * @param options - Optional request options
   * @returns The list of customer entitlements
   */
  public async list(
    customerIdOrKey: operations['listCustomerEntitlementsV2']['parameters']['path']['customerIdOrKey'],
    query?: operations['listCustomerEntitlementsV2']['parameters']['query'],
    options?: RequestOptions
  ) {
    const resp = await this.client.GET(
      '/api/v2/customers/{customerIdOrKey}/entitlements',
      {
        params: {
          path: { customerIdOrKey },
          query,
        },
        ...options,
      }
    )

    return transformResponse(resp)
  }

  /**
   * Create a customer entitlement
   * @param customerIdOrKey - The ID or Key of the customer
   * @param entitlement - The entitlement to create
   * @param options - Optional request options
   * @returns The created entitlement
   */
  public async create(
    customerIdOrKey: operations['createCustomerEntitlementV2']['parameters']['path']['customerIdOrKey'],
    entitlement: operations['createCustomerEntitlementV2']['requestBody']['content']['application/json'],
    options?: RequestOptions
  ) {
    const resp = await this.client.POST(
      '/api/v2/customers/{customerIdOrKey}/entitlements',
      {
        body: entitlement,
        params: {
          path: { customerIdOrKey },
        },
        ...options,
      }
    )

    return transformResponse(resp)
  }

  /**
   * Get a customer entitlement
   * @param customerIdOrKey - The ID or Key of the customer
   * @param entitlementIdOrFeatureKey - The ID or feature key of the entitlement
   * @param options - Optional request options
   * @returns The customer entitlement
   */
  public async get(
    customerIdOrKey: operations['getCustomerEntitlementV2']['parameters']['path']['customerIdOrKey'],
    entitlementIdOrFeatureKey: operations['getCustomerEntitlementV2']['parameters']['path']['entitlementIdOrFeatureKey'],
    options?: RequestOptions
  ) {
    const resp = await this.client.GET(
      '/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}',
      {
        params: {
          path: { customerIdOrKey, entitlementIdOrFeatureKey },
        },
        ...options,
      }
    )

    return transformResponse(resp)
  }

  /**
   * Delete a customer entitlement
   * @param customerIdOrKey - The ID or Key of the customer
   * @param entitlementIdOrFeatureKey - The ID or feature key of the entitlement
   * @param options - Optional request options
   * @returns The deleted entitlement
   */
  public async delete(
    customerIdOrKey: operations['deleteCustomerEntitlementV2']['parameters']['path']['customerIdOrKey'],
    entitlementIdOrFeatureKey: operations['deleteCustomerEntitlementV2']['parameters']['path']['entitlementIdOrFeatureKey'],
    options?: RequestOptions
  ) {
    const resp = await this.client.DELETE(
      '/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}',
      {
        params: {
          path: { customerIdOrKey, entitlementIdOrFeatureKey },
        },
        ...options,
      }
    )

    return transformResponse(resp)
  }

  /**
   * Get customer entitlement value (v2)
   * @param customerIdOrKey - The ID or Key of the customer
   * @param entitlementIdOrFeatureKey - The ID or feature key of the entitlement
   * @param query - The query parameters
   * @param options - Optional request options
   * @returns The entitlement value
   */
  public async value(
    customerIdOrKey: operations['getCustomerEntitlementValueV2']['parameters']['path']['customerIdOrKey'],
    entitlementIdOrFeatureKey: operations['getCustomerEntitlementValueV2']['parameters']['path']['entitlementIdOrFeatureKey'],
    query?: operations['getCustomerEntitlementValueV2']['parameters']['query'],
    options?: RequestOptions
  ) {
    const resp = await this.client.GET(
      '/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/value',
      {
        params: {
          path: { customerIdOrKey, entitlementIdOrFeatureKey },
          query,
        },
        ...options,
      }
    )

    return transformResponse(resp)
  }

  /**
   * List customer entitlement grants
   * @param customerIdOrKey - The ID or Key of the customer
   * @param entitlementIdOrFeatureKey - The ID or feature key of the entitlement
   * @param query - The query parameters
   * @param options - Optional request options
   * @returns The list of grants
   */
  public async listGrants(
    customerIdOrKey: operations['listCustomerEntitlementGrantsV2']['parameters']['path']['customerIdOrKey'],
    entitlementIdOrFeatureKey: operations['listCustomerEntitlementGrantsV2']['parameters']['path']['entitlementIdOrFeatureKey'],
    query?: operations['listCustomerEntitlementGrantsV2']['parameters']['query'],
    options?: RequestOptions
  ) {
    const resp = await this.client.GET(
      '/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants',
      {
        params: {
          path: { customerIdOrKey, entitlementIdOrFeatureKey },
          query,
        },
        ...options,
      }
    )

    return transformResponse(resp)
  }

  /**
   * Create customer entitlement grant
   * @param customerIdOrKey - The ID or Key of the customer
   * @param entitlementIdOrFeatureKey - The ID or feature key of the entitlement
   * @param grant - The grant to create
   * @param options - Optional request options
   * @returns The created grant
   */
  public async createGrant(
    customerIdOrKey: operations['createCustomerEntitlementGrantV2']['parameters']['path']['customerIdOrKey'],
    entitlementIdOrFeatureKey: operations['createCustomerEntitlementGrantV2']['parameters']['path']['entitlementIdOrFeatureKey'],
    grant: operations['createCustomerEntitlementGrantV2']['requestBody']['content']['application/json'],
    options?: RequestOptions
  ) {
    const resp = await this.client.POST(
      '/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants',
      {
        body: grant,
        params: {
          path: { customerIdOrKey, entitlementIdOrFeatureKey },
        },
        ...options,
      }
    )

    return transformResponse(resp)
  }
}
