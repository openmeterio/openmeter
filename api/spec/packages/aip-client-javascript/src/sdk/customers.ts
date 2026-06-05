import { type Client } from '../core.js'
import { unwrap, type RequestOptions } from '../lib/types.js'
import {
  createCustomer,
  getCustomer,
  listCustomers,
  upsertCustomer,
  deleteCustomer,
  getCustomerBilling,
  updateCustomerBilling,
  updateCustomerBillingAppData,
  createCustomerStripeCheckoutSession,
  createCustomerStripePortalSession,
  createCreditGrant,
  getCreditGrant,
  listCreditGrants,
  getCustomerCreditBalance,
  createCreditAdjustment,
  listCreditTransactions,
  listCustomerCharges,
} from '../funcs/customers.js'
import type {
  CreateCustomerRequest,
  CreateCustomerResponse,
  GetCustomerRequest,
  GetCustomerResponse,
  ListCustomersRequest,
  ListCustomersResponse,
  UpsertCustomerRequest,
  UpsertCustomerResponse,
  DeleteCustomerRequest,
  DeleteCustomerResponse,
  GetCustomerBillingRequest,
  GetCustomerBillingResponse,
  UpdateCustomerBillingRequest,
  UpdateCustomerBillingResponse,
  UpdateCustomerBillingAppDataRequest,
  UpdateCustomerBillingAppDataResponse,
  CreateCustomerStripeCheckoutSessionRequest,
  CreateCustomerStripeCheckoutSessionResponse,
  CreateCustomerStripePortalSessionRequest,
  CreateCustomerStripePortalSessionResponse,
  CreateCreditGrantRequest,
  CreateCreditGrantResponse,
  GetCreditGrantRequest,
  GetCreditGrantResponse,
  ListCreditGrantsRequest,
  ListCreditGrantsResponse,
  GetCustomerCreditBalanceRequest,
  GetCustomerCreditBalanceResponse,
  CreateCreditAdjustmentRequest,
  CreateCreditAdjustmentResponse,
  ListCreditTransactionsRequest,
  ListCreditTransactionsResponse,
  ListCustomerChargesRequest,
  ListCustomerChargesResponse,
} from '../models/operations/customers.js'

export class Customers {
  constructor(private readonly _client: Client) {}

  async create(
    request: CreateCustomerRequest,
    options?: RequestOptions,
  ): Promise<CreateCustomerResponse> {
    return unwrap(await createCustomer(this._client, request, options))
  }

  async get(
    request: GetCustomerRequest,
    options?: RequestOptions,
  ): Promise<GetCustomerResponse> {
    return unwrap(await getCustomer(this._client, request, options))
  }

  async list(
    request?: ListCustomersRequest,
    options?: RequestOptions,
  ): Promise<ListCustomersResponse> {
    return unwrap(await listCustomers(this._client, request, options))
  }

  async upsert(
    request: UpsertCustomerRequest,
    options?: RequestOptions,
  ): Promise<UpsertCustomerResponse> {
    return unwrap(await upsertCustomer(this._client, request, options))
  }

  async delete(
    request: DeleteCustomerRequest,
    options?: RequestOptions,
  ): Promise<DeleteCustomerResponse> {
    return unwrap(await deleteCustomer(this._client, request, options))
  }

  private _billing?: CustomersBilling
  get billing(): CustomersBilling {
    return (this._billing ??= new CustomersBilling(this._client))
  }

  private _credits?: CustomersCredits
  get credits(): CustomersCredits {
    return (this._credits ??= new CustomersCredits(this._client))
  }

  private _charges?: CustomersCharges
  get charges(): CustomersCharges {
    return (this._charges ??= new CustomersCharges(this._client))
  }
}

export class CustomersBilling {
  constructor(private readonly _client: Client) {}

  async get(
    request: GetCustomerBillingRequest,
    options?: RequestOptions,
  ): Promise<GetCustomerBillingResponse> {
    return unwrap(await getCustomerBilling(this._client, request, options))
  }

  async update(
    request: UpdateCustomerBillingRequest,
    options?: RequestOptions,
  ): Promise<UpdateCustomerBillingResponse> {
    return unwrap(await updateCustomerBilling(this._client, request, options))
  }

  async updateAppData(
    request: UpdateCustomerBillingAppDataRequest,
    options?: RequestOptions,
  ): Promise<UpdateCustomerBillingAppDataResponse> {
    return unwrap(await updateCustomerBillingAppData(this._client, request, options))
  }

  async createStripeCheckoutSession(
    request: CreateCustomerStripeCheckoutSessionRequest,
    options?: RequestOptions,
  ): Promise<CreateCustomerStripeCheckoutSessionResponse> {
    return unwrap(await createCustomerStripeCheckoutSession(this._client, request, options))
  }

  async createStripePortalSession(
    request: CreateCustomerStripePortalSessionRequest,
    options?: RequestOptions,
  ): Promise<CreateCustomerStripePortalSessionResponse> {
    return unwrap(await createCustomerStripePortalSession(this._client, request, options))
  }
}

export class CustomersCredits {
  constructor(private readonly _client: Client) {}

  private _grants?: CustomersCreditsGrants
  get grants(): CustomersCreditsGrants {
    return (this._grants ??= new CustomersCreditsGrants(this._client))
  }

  private _balance?: CustomersCreditsBalance
  get balance(): CustomersCreditsBalance {
    return (this._balance ??= new CustomersCreditsBalance(this._client))
  }

  private _adjustments?: CustomersCreditsAdjustments
  get adjustments(): CustomersCreditsAdjustments {
    return (this._adjustments ??= new CustomersCreditsAdjustments(this._client))
  }

  private _transactions?: CustomersCreditsTransactions
  get transactions(): CustomersCreditsTransactions {
    return (this._transactions ??= new CustomersCreditsTransactions(this._client))
  }
}

export class CustomersCreditsGrants {
  constructor(private readonly _client: Client) {}

  async create(
    request: CreateCreditGrantRequest,
    options?: RequestOptions,
  ): Promise<CreateCreditGrantResponse> {
    return unwrap(await createCreditGrant(this._client, request, options))
  }

  async get(
    request: GetCreditGrantRequest,
    options?: RequestOptions,
  ): Promise<GetCreditGrantResponse> {
    return unwrap(await getCreditGrant(this._client, request, options))
  }

  async list(
    request: ListCreditGrantsRequest,
    options?: RequestOptions,
  ): Promise<ListCreditGrantsResponse> {
    return unwrap(await listCreditGrants(this._client, request, options))
  }
}

export class CustomersCreditsBalance {
  constructor(private readonly _client: Client) {}

  async get(
    request: GetCustomerCreditBalanceRequest,
    options?: RequestOptions,
  ): Promise<GetCustomerCreditBalanceResponse> {
    return unwrap(await getCustomerCreditBalance(this._client, request, options))
  }
}

export class CustomersCreditsAdjustments {
  constructor(private readonly _client: Client) {}

  async create(
    request: CreateCreditAdjustmentRequest,
    options?: RequestOptions,
  ): Promise<CreateCreditAdjustmentResponse> {
    return unwrap(await createCreditAdjustment(this._client, request, options))
  }
}

export class CustomersCreditsTransactions {
  constructor(private readonly _client: Client) {}

  async list(
    request: ListCreditTransactionsRequest,
    options?: RequestOptions,
  ): Promise<ListCreditTransactionsResponse> {
    return unwrap(await listCreditTransactions(this._client, request, options))
  }
}

export class CustomersCharges {
  constructor(private readonly _client: Client) {}

  async list(
    request: ListCustomerChargesRequest,
    options?: RequestOptions,
  ): Promise<ListCustomerChargesResponse> {
    return unwrap(await listCustomerCharges(this._client, request, options))
  }
}
