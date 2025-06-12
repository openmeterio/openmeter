import { transformResponse } from './utils.js'
import type { RequestOptions } from './common.js'
import type {
  CustomerAppData,
  CustomerCreate,
  CustomerReplaceUpdate,
  operations,
  paths,
} from './schemas.js'
import type { Client } from 'openapi-fetch'

/**
 * Customers
 * Manage customer subscription lifecycles and plan assignments.
 */
export class Customers {
  public apps: CustomerApps
  public entitlements: CustomerEntitlements

  constructor(private client: Client<paths, `${string}/${string}`>) {
    this.apps = new CustomerApps(client)
    this.entitlements = new CustomerEntitlements(client)
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
}
