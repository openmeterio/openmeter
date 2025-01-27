import { transformResponse, type RequestOptions } from './utils.js'
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
   * @param id - The ID of the customer
   * @param signal - An optional abort signal
   * @returns The customer
   */
  public async get(
    id: operations['getCustomer']['parameters']['path']['id'],
    options?: RequestOptions
  ) {
    const resp = await this.client.GET('/api/v1/customers/{id}', {
      params: {
        path: {
          id,
        },
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Update a customer
   * @param id - The ID of the customer
   * @param customer - The customer to update
   * @param signal - An optional abort signal
   * @returns The updated customer
   */
  public async update(
    id: operations['updateCustomer']['parameters']['path']['id'],
    customer: CustomerReplaceUpdate,
    options?: RequestOptions
  ) {
    const resp = await this.client.PUT('/api/v1/customers/{id}', {
      body: customer,
      params: {
        path: {
          id,
        },
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Delete a customer
   * @param id - The ID of the customer
   * @param signal - An optional abort signal
   * @returns The deleted customer
   */
  public async delete(
    id: operations['deleteCustomer']['parameters']['path']['id'],
    options?: RequestOptions
  ) {
    const resp = await this.client.DELETE('/api/v1/customers/{id}', {
      params: {
        path: {
          id,
        },
      },
      ...options,
    })

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
}

/**
 * Customer Apps
 * Manage customer apps.
 */
export class CustomerApps {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Upsert customer app data
   * @param customerId - The ID of the customer
   * @param appData - The app data to upsert
   * @param signal - An optional abort signal
   * @returns The upserted app data
   */
  public async upsert(
    customerId: operations['upsertCustomerAppData']['parameters']['path']['customerId'],
    appData: CustomerAppData[],
    options?: RequestOptions
  ) {
    const resp = await this.client.PUT('/api/v1/customers/{customerId}/apps', {
      body: appData,
      params: {
        path: {
          customerId,
        },
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List customer app data
   * @param customerId - The ID of the customer
   * @param query - The query parameters
   * @param signal - An optional abort signal
   * @returns The list of customer app data
   */
  public async list(
    customerId: operations['listCustomerAppData']['parameters']['path']['customerId'],
    query?: operations['listCustomerAppData']['parameters']['query'],
    options?: RequestOptions
  ) {
    const resp = await this.client.GET('/api/v1/customers/{customerId}/apps', {
      params: {
        path: { customerId },
        query,
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Delete customer app data
   * @param customerId - The ID of the customer
   * @param appId - The ID of the app
   * @param signal - An optional abort signal
   * @returns The deleted customer app data
   */
  public async delete(
    customerId: operations['deleteCustomerAppData']['parameters']['path']['customerId'],
    appId: operations['deleteCustomerAppData']['parameters']['path']['appId'],
    options?: RequestOptions
  ) {
    const resp = await this.client.DELETE(
      '/api/v1/customers/{customerId}/apps/{appId}',
      {
        params: { path: { appId, customerId } },
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
   * @param customerId - The ID of the customer
   * @param featureKey - The key of the feature
   * @param signal - An optional abort signal
   * @returns The value of the entitlement
   */
  public async value(
    customerId: operations['getCustomerEntitlementValue']['parameters']['path']['customerId'],
    featureKey: operations['getCustomerEntitlementValue']['parameters']['path']['featureKey'],
    options?: RequestOptions
  ) {
    const resp = await this.client.GET(
      '/api/v1/customers/{customerId}/entitlements/{featureKey}/value',
      {
        params: { path: { customerId, featureKey } },
        ...options,
      }
    )

    return transformResponse(resp)
  }
}
