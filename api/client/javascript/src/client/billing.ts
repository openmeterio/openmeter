import type { Client } from 'openapi-fetch'
import type { RequestOptions } from './common.js'
import type {
  BillingProfileCreate,
  BillingProfileCustomerOverrideCreate,
  BillingProfileReplaceUpdateWithWorkflow,
  InvoicePendingLineCreateInput,
  InvoiceReplaceUpdate,
  InvoiceSimulationInput,
  operations,
  paths,
  VoidInvoiceActionInput,
} from './schemas.js'
import { transformResponse } from './utils.js'
/**
 * Billing
 */
export class Billing {
  public profiles: BillingProfiles
  public invoices: BillingInvoices
  public customers: BillingCustomers

  constructor(private client: Client<paths, `${string}/${string}`>) {
    this.profiles = new BillingProfiles(this.client)
    this.invoices = new BillingInvoices(this.client)
    this.customers = new BillingCustomers(this.client)
  }
}

/**
 * Billing Profiles
 */
export class BillingProfiles {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Create a billing profile
   * @param billingProfile - The billing profile to create
   * @param signal - An optional abort signal
   * @returns The created billing profile
   */
  public async create(
    billingProfile: BillingProfileCreate,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST('/api/v1/billing/profiles', {
      body: billingProfile,
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get a billing profile by ID
   * @param id - The ID of the billing profile to get
   * @param signal - An optional abort signal
   * @returns The billing profile
   */
  public async get(
    id: operations['getBillingProfile']['parameters']['path']['id'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/billing/profiles/{id}', {
      params: {
        path: { id },
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List billing profiles
   * @param query - The query parameters
   * @param signal - An optional abort signal
   * @returns The billing profiles
   */
  public async list(
    query?: operations['listBillingProfiles']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/billing/profiles', {
      params: {
        query,
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Update a billing profile
   * @param id - The ID of the billing profile to update
   * @param billingProfile - The billing profile to update
   * @param signal - An optional abort signal
   * @returns The updated billing profile
   */
  public async update(
    id: operations['updateBillingProfile']['parameters']['path']['id'],
    billingProfile: BillingProfileReplaceUpdateWithWorkflow,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT('/api/v1/billing/profiles/{id}', {
      body: billingProfile,
      params: {
        path: { id },
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Delete a billing profile
   * @param id - The ID of the billing profile to delete
   * @param options - The request options
   * @returns The deleted billing profile
   */
  public async delete(
    id: operations['deleteBillingProfile']['parameters']['path']['id'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.DELETE('/api/v1/billing/profiles/{id}', {
      params: {
        path: { id },
      },
      ...options,
    })

    return transformResponse(resp)
  }
}

/**
 * Billing Invoices
 */
export class BillingInvoices {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * List invoices
   * @param query - The query parameters
   * @param signal - An optional abort signal
   * @returns The invoices
   */
  public async list(
    query?: operations['listInvoices']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/billing/invoices', {
      params: {
        query,
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get an invoice by ID
   * @param id - The ID of the invoice to get
   * @param signal - An optional abort signal
   * @returns The invoice
   */
  public async get(
    id: operations['getInvoice']['parameters']['path']['invoiceId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/billing/invoices/{invoiceId}', {
      params: {
        path: { invoiceId: id },
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Update an invoice
   * @description Only invoices in draft or earlier status can be updated.
   * @param id - The ID of the invoice to update
   * @param invoice - The invoice to update
   * @param signal - An optional abort signal
   * @returns The updated invoice
   */
  public async update(
    id: operations['updateInvoice']['parameters']['path']['invoiceId'],
    invoice: InvoiceReplaceUpdate,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT('/api/v1/billing/invoices/{invoiceId}', {
      body: invoice,
      params: { path: { invoiceId: id } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Delete an invoice
   * @description Only invoices that are in the draft (or earlier) status can be deleted.
   * @param id - The ID of the invoice to delete
   * @param options - The request options
   * @returns The deleted invoice
   */
  public async delete(
    id: operations['deleteInvoice']['parameters']['path']['invoiceId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.DELETE(
      '/api/v1/billing/invoices/{invoiceId}',
      {
        params: { path: { invoiceId: id } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Advance the invoice to the next status
   * @description The call doesn't "approve the invoice", it only advances the invoice to the next status if the transition would be automatic. The action can be called when the invoice's statusDetails' actions field contain the "advance" action.
   * @param id - The ID of the invoice to advance
   * @param signal - An optional abort signal
   * @returns The advanced invoice
   */
  public async advance(
    id: operations['advanceInvoiceAction']['parameters']['path']['invoiceId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/api/v1/billing/invoices/{invoiceId}/advance',
      {
        params: { path: { invoiceId: id } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Approve an invoice
   * @description This call instantly sends the invoice to the customer using the configured billing profile app.
   * @param id - The ID of the invoice to approve
   * @param signal - An optional abort signal
   * @returns The approved invoice
   */
  public async approve(
    id: operations['approveInvoiceAction']['parameters']['path']['invoiceId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/api/v1/billing/invoices/{invoiceId}/approve',
      {
        params: { path: { invoiceId: id } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Retry advancing the invoice after a failed attempt.
   * @param id - The ID of the invoice to retry
   * @param signal - An optional abort signal
   * @returns The retried invoice
   */
  public async retry(
    id: operations['retryInvoiceAction']['parameters']['path']['invoiceId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/api/v1/billing/invoices/{invoiceId}/retry',
      {
        params: { path: { invoiceId: id } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Void an invoice
   * @description Void an invoice
   *
   *     Only invoices that have been alread issued can be voided.
   * @param id - The ID of the invoice to void
   * @param signal - An optional abort signal
   * @returns The voided invoice
   */
  public async void(
    id: operations['voidInvoiceAction']['parameters']['path']['invoiceId'],
    body: VoidInvoiceActionInput,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/api/v1/billing/invoices/{invoiceId}/void',
      {
        body,
        params: { path: { invoiceId: id } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Recalculate an invoice's tax amounts
   * @param id - The ID of the invoice to recalculate
   * @param signal - An optional abort signal
   * @returns The recalculated invoice
   */
  public async recalculateTax(
    id: operations['recalculateInvoiceTaxAction']['parameters']['path']['invoiceId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/api/v1/billing/invoices/{invoiceId}/taxes/recalculate',
      {
        params: { path: { invoiceId: id } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Snapshot invoice line item quantities
   * @description Snapshot the quantities of the invoice line items. This is useful for invoices that have usage-based line items.
   * @param id - The ID of the invoice to snapshot
   * @param signal - An optional abort signal
   * @returns The invoice with snapshotted quantities
   */
  public async snapshotQuantities(
    id: operations['snapshotQuantitiesInvoiceAction']['parameters']['path']['invoiceId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/api/v1/billing/invoices/{invoiceId}/snapshot-quantities',
      {
        params: { path: { invoiceId: id } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Simulate an invoice for a customer
   * @param id - The ID of the customer to simulate the invoice for
   * @param signal - An optional abort signal
   * @returns The simulated invoice
   */
  public async simulate(
    id: operations['simulateInvoice']['parameters']['path']['customerId'],
    body: InvoiceSimulationInput,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/api/v1/billing/customers/{customerId}/invoices/simulate',
      {
        body,
        params: { path: { customerId: id } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Create pending line items
   * @description Create new pending line items (charges).
   *     This call is used to create a new pending line item for the customer if required a new
   *     gathering invoice will be created.
   *
   *     A new invoice will be created if:
   *     - there is no invoice in gathering state
   *     - the currency of the line item doesn't match the currency of any invoices in gathering state
   * @param customerId - The ID of the customer to create the line items for
   * @param body - The line items to create
   * @param signal - An optional abort signal
   * @returns The created line items
   */
  public async createLineItems(
    customerId: operations['createPendingInvoiceLine']['parameters']['path']['customerId'],
    body: InvoicePendingLineCreateInput,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/api/v1/billing/customers/{customerId}/invoices/pending-lines',
      {
        body,
        params: { path: { customerId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Invoice a customer based on the pending line items
   * @description Create a new invoice from the pending line items. This should only be called if for some reason we need to invoice a customer outside of the normal billing cycle.
   * @param body - The invoice data
   * @param options - The request options
   * @returns The created invoices
   */
  public async invoicePendingLines(
    body: operations['invoicePendingLinesAction']['requestBody']['content']['application/json'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST('/api/v1/billing/invoices/invoice', {
      body,
      ...options,
    })

    return transformResponse(resp)
  }
}

/**
 * Billing Customer Invoices and Overrides
 */
export class BillingCustomers {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Create or update a customer override
   * @param id - The ID of the customer to create the override for
   * @param body - The customer override to create
   * @param signal - An optional abort signal
   * @returns The created customer override
   */
  public async createOverride(
    id: operations['upsertBillingProfileCustomerOverride']['parameters']['path']['customerId'],
    body: BillingProfileCustomerOverrideCreate,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT(
      '/api/v1/billing/customers/{customerId}',
      {
        body,
        params: { path: { customerId: id } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Get a customer override
   * @param id - The ID of the customer to get the override for
   * @param signal - An optional abort signal
   * @returns The customer override
   */
  public async getOverride(
    id: operations['getBillingProfileCustomerOverride']['parameters']['path']['customerId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/api/v1/billing/customers/{customerId}',
      {
        params: { path: { customerId: id } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * List customer overrides
   * @param query - The query parameters
   * @param signal - An optional abort signal
   * @returns The customer overrides
   */
  public async listOverrides(
    query?: operations['listBillingProfileCustomerOverrides']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/billing/customers', {
      params: { query },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Delete a customer override
   * @param id - The ID of the customer to delete the override for
   * @param signal - An optional abort signal
   * @returns The deleted customer override
   */
  public async deleteOverride(
    id: operations['deleteBillingProfileCustomerOverride']['parameters']['path']['customerId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.DELETE(
      '/api/v1/billing/customers/{customerId}',
      {
        params: { path: { customerId: id } },
        ...options,
      },
    )

    return transformResponse(resp)
  }
}
