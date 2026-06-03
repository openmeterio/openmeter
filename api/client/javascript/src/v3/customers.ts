import type { Client } from 'openapi-fetch'
import type { RequestOptions } from '../client/common.js'
import { transformResponse } from '../client/utils.js'
import type {
  BillingCustomerStripeCreateCheckoutSessionRequest,
  BillingCustomerStripeCreateCustomerPortalSessionRequest,
  CreateCreditAdjustmentRequest,
  CreateCreditGrantRequest,
  CreateCustomerRequest,
  operations,
  paths,
  UpdateCreditGrantExternalSettlementRequest,
  UpsertAppCustomerDataRequest,
  UpsertCustomerBillingDataRequest,
  UpsertCustomerRequest,
} from './schemas.js'

/**
 * Customers (v3)
 *
 * Thin wrapper over the v3 customers endpoints. Bodies use the v3 wire shape
 * verbatim (snake_case); no field renaming (Option A).
 */
export class Customers {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Create a customer
   */
  public async create(
    customer: CreateCustomerRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST('/openmeter/customers', {
      body: customer,
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List customers
   */
  public async list(
    params?: operations['list-customers']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/openmeter/customers', {
      params: { query: params },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get a customer by ID
   */
  public async get(customerId: string, options?: RequestOptions) {
    const resp = await this.client.GET('/openmeter/customers/{customerId}', {
      params: { path: { customerId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Upsert (replace) a customer
   */
  public async upsert(
    customerId: string,
    customer: UpsertCustomerRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT('/openmeter/customers/{customerId}', {
      body: customer,
      params: { path: { customerId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Delete a customer by ID
   */
  public async delete(customerId: string, options?: RequestOptions) {
    const resp = await this.client.DELETE('/openmeter/customers/{customerId}', {
      params: { path: { customerId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List credit grants for a customer
   */
  public async listCreditGrants(
    customerId: string,
    params?: operations['list-credit-grants']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/openmeter/customers/{customerId}/credits/grants',
      {
        params: { path: { customerId }, query: params },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Create a credit grant for a customer
   */
  public async createCreditGrant(
    customerId: string,
    body: CreateCreditGrantRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/openmeter/customers/{customerId}/credits/grants',
      {
        body,
        params: { path: { customerId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Get a credit grant for a customer
   */
  public async getCreditGrant(
    customerId: string,
    creditGrantId: string,
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/openmeter/customers/{customerId}/credits/grants/{creditGrantId}',
      {
        params: { path: { creditGrantId, customerId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * List credit transactions for a customer
   */
  public async listCreditTransactions(
    customerId: string,
    params?: operations['list-credit-transactions']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/openmeter/customers/{customerId}/credits/transactions',
      {
        params: { path: { customerId }, query: params },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Create a credit adjustment for a customer
   */
  public async createCreditAdjustment(
    customerId: string,
    body: CreateCreditAdjustmentRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/openmeter/customers/{customerId}/credits/adjustments',
      {
        body,
        params: { path: { customerId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Get a customer's credit balance
   */
  public async getCreditBalance(
    customerId: string,
    params?: operations['get-customer-credit-balance']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/openmeter/customers/{customerId}/credits/balance',
      {
        params: { path: { customerId }, query: params },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Update the external settlement status of a credit grant
   */
  public async updateCreditGrantExternalSettlement(
    customerId: string,
    creditGrantId: string,
    body: UpdateCreditGrantExternalSettlementRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/openmeter/customers/{customerId}/credits/grants/{creditGrantId}/settlement/external',
      {
        body,
        params: { path: { creditGrantId, customerId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Get customer billing data
   */
  public async getBilling(customerId: string, options?: RequestOptions) {
    const resp = await this.client.GET(
      '/openmeter/customers/{customerId}/billing',
      {
        params: { path: { customerId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Update customer billing data
   */
  public async updateBilling(
    customerId: string,
    body: UpsertCustomerBillingDataRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT(
      '/openmeter/customers/{customerId}/billing',
      {
        body,
        params: { path: { customerId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Update customer billing app data
   */
  public async updateBillingAppData(
    customerId: string,
    body: UpsertAppCustomerDataRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT(
      '/openmeter/customers/{customerId}/billing/app-data',
      {
        body,
        params: { path: { customerId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Create a Stripe Checkout Session for the customer
   */
  public async createStripeCheckoutSession(
    customerId: string,
    body: BillingCustomerStripeCreateCheckoutSessionRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/openmeter/customers/{customerId}/billing/stripe/checkout-sessions',
      {
        body,
        params: { path: { customerId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Create a Stripe Customer Portal Session for the customer
   */
  public async createStripePortalSession(
    customerId: string,
    body: BillingCustomerStripeCreateCustomerPortalSessionRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/openmeter/customers/{customerId}/billing/stripe/portal-sessions',
      {
        body,
        params: { path: { customerId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * List customer charges
   */
  public async listCharges(
    customerId: string,
    params?: operations['list-customer-charges']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/openmeter/customers/{customerId}/charges',
      {
        params: { path: { customerId }, query: params },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * List the customer's active features and their access
   */
  public async listEntitlementAccess(
    customerId: string,
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/openmeter/customers/{customerId}/entitlement-access',
      {
        params: { path: { customerId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }
}
