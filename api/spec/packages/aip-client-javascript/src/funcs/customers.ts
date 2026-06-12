import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { encodePath, toURLSearchParams, encodeSort } from '../lib/encodings.js'
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
  UpdateCreditGrantExternalSettlementRequest,
  UpdateCreditGrantExternalSettlementResponse,
  ListCreditTransactionsRequest,
  ListCreditTransactionsResponse,
  ListCustomerChargesRequest,
  ListCustomerChargesResponse,
  CreateCustomerChargesRequest,
  CreateCustomerChargesResponse,
} from '../models/operations/customers.js'

export function createCustomer(
  client: Client,
  req: CreateCustomerRequest,
  options?: RequestOptions,
): Promise<Result<CreateCustomerResponse>> {
  return request(() =>
    http(client)
      .post('openmeter/customers', { ...options, json: req })
      .json<CreateCustomerResponse>(),
  )
}

export function getCustomer(
  client: Client,
  req: GetCustomerRequest,
  options?: RequestOptions,
): Promise<Result<GetCustomerResponse>> {
  const path = encodePath('openmeter/customers/{customerId}', {
    customerId: req.customerId,
  })
  return request(() =>
    http(client).get(path, options).json<GetCustomerResponse>(),
  )
}

export function listCustomers(
  client: Client,
  req: ListCustomersRequest = {},
  options?: RequestOptions,
): Promise<Result<ListCustomersResponse>> {
  const searchParams = toURLSearchParams({
    page: req.page,
    sort: encodeSort(req.sort),
    filter: req.filter,
  })
  return request(() =>
    http(client)
      .get('openmeter/customers', { ...options, searchParams })
      .json<ListCustomersResponse>(),
  )
}

export function upsertCustomer(
  client: Client,
  req: UpsertCustomerRequest,
  options?: RequestOptions,
): Promise<Result<UpsertCustomerResponse>> {
  const path = encodePath('openmeter/customers/{customerId}', {
    customerId: req.customerId,
  })
  return request(() =>
    http(client)
      .put(path, { ...options, json: req.body })
      .json<UpsertCustomerResponse>(),
  )
}

export function deleteCustomer(
  client: Client,
  req: DeleteCustomerRequest,
  options?: RequestOptions,
): Promise<Result<DeleteCustomerResponse>> {
  const path = encodePath('openmeter/customers/{customerId}', {
    customerId: req.customerId,
  })
  return request(async () => {
    await http(client).delete(path, options)
  })
}

export function getCustomerBilling(
  client: Client,
  req: GetCustomerBillingRequest,
  options?: RequestOptions,
): Promise<Result<GetCustomerBillingResponse>> {
  const path = encodePath('openmeter/customers/{customerId}/billing', {
    customerId: req.customerId,
  })
  return request(() =>
    http(client).get(path, options).json<GetCustomerBillingResponse>(),
  )
}

export function updateCustomerBilling(
  client: Client,
  req: UpdateCustomerBillingRequest,
  options?: RequestOptions,
): Promise<Result<UpdateCustomerBillingResponse>> {
  const path = encodePath('openmeter/customers/{customerId}/billing', {
    customerId: req.customerId,
  })
  return request(() =>
    http(client)
      .put(path, { ...options, json: req.body })
      .json<UpdateCustomerBillingResponse>(),
  )
}

export function updateCustomerBillingAppData(
  client: Client,
  req: UpdateCustomerBillingAppDataRequest,
  options?: RequestOptions,
): Promise<Result<UpdateCustomerBillingAppDataResponse>> {
  const path = encodePath('openmeter/customers/{customerId}/billing/app-data', {
    customerId: req.customerId,
  })
  return request(() =>
    http(client)
      .put(path, { ...options, json: req.body })
      .json<UpdateCustomerBillingAppDataResponse>(),
  )
}

export function createCustomerStripeCheckoutSession(
  client: Client,
  req: CreateCustomerStripeCheckoutSessionRequest,
  options?: RequestOptions,
): Promise<Result<CreateCustomerStripeCheckoutSessionResponse>> {
  const path = encodePath(
    'openmeter/customers/{customerId}/billing/stripe/checkout-sessions',
    { customerId: req.customerId },
  )
  return request(() =>
    http(client)
      .post(path, { ...options, json: req.body })
      .json<CreateCustomerStripeCheckoutSessionResponse>(),
  )
}

export function createCustomerStripePortalSession(
  client: Client,
  req: CreateCustomerStripePortalSessionRequest,
  options?: RequestOptions,
): Promise<Result<CreateCustomerStripePortalSessionResponse>> {
  const path = encodePath(
    'openmeter/customers/{customerId}/billing/stripe/portal-sessions',
    { customerId: req.customerId },
  )
  return request(() =>
    http(client)
      .post(path, { ...options, json: req.body })
      .json<CreateCustomerStripePortalSessionResponse>(),
  )
}

export function createCreditGrant(
  client: Client,
  req: CreateCreditGrantRequest,
  options?: RequestOptions,
): Promise<Result<CreateCreditGrantResponse>> {
  const path = encodePath('openmeter/customers/{customerId}/credits/grants', {
    customerId: req.customerId,
  })
  return request(() =>
    http(client)
      .post(path, { ...options, json: req.body })
      .json<CreateCreditGrantResponse>(),
  )
}

export function getCreditGrant(
  client: Client,
  req: GetCreditGrantRequest,
  options?: RequestOptions,
): Promise<Result<GetCreditGrantResponse>> {
  const path = encodePath(
    'openmeter/customers/{customerId}/credits/grants/{creditGrantId}',
    { customerId: req.customerId, creditGrantId: req.creditGrantId },
  )
  return request(() =>
    http(client).get(path, options).json<GetCreditGrantResponse>(),
  )
}

export function listCreditGrants(
  client: Client,
  req: ListCreditGrantsRequest,
  options?: RequestOptions,
): Promise<Result<ListCreditGrantsResponse>> {
  const searchParams = toURLSearchParams({
    page: req.page,
    filter: req.filter,
  })
  const path = encodePath('openmeter/customers/{customerId}/credits/grants', {
    customerId: req.customerId,
  })
  return request(() =>
    http(client)
      .get(path, { ...options, searchParams })
      .json<ListCreditGrantsResponse>(),
  )
}

export function getCustomerCreditBalance(
  client: Client,
  req: GetCustomerCreditBalanceRequest,
  options?: RequestOptions,
): Promise<Result<GetCustomerCreditBalanceResponse>> {
  const searchParams = toURLSearchParams({
    timestamp: req.timestamp,
    filter: req.filter,
  })
  const path = encodePath('openmeter/customers/{customerId}/credits/balance', {
    customerId: req.customerId,
  })
  return request(() =>
    http(client)
      .get(path, { ...options, searchParams })
      .json<GetCustomerCreditBalanceResponse>(),
  )
}

export function createCreditAdjustment(
  client: Client,
  req: CreateCreditAdjustmentRequest,
  options?: RequestOptions,
): Promise<Result<CreateCreditAdjustmentResponse>> {
  const path = encodePath(
    'openmeter/customers/{customerId}/credits/adjustments',
    { customerId: req.customerId },
  )
  return request(() =>
    http(client)
      .post(path, { ...options, json: req.body })
      .json<CreateCreditAdjustmentResponse>(),
  )
}

export function updateCreditGrantExternalSettlement(
  client: Client,
  req: UpdateCreditGrantExternalSettlementRequest,
  options?: RequestOptions,
): Promise<Result<UpdateCreditGrantExternalSettlementResponse>> {
  const path = encodePath(
    'openmeter/customers/{customerId}/credits/grants/{creditGrantId}/settlement/external',
    { customerId: req.customerId, creditGrantId: req.creditGrantId },
  )
  return request(() =>
    http(client)
      .post(path, { ...options, json: req.body })
      .json<UpdateCreditGrantExternalSettlementResponse>(),
  )
}

export function listCreditTransactions(
  client: Client,
  req: ListCreditTransactionsRequest,
  options?: RequestOptions,
): Promise<Result<ListCreditTransactionsResponse>> {
  const searchParams = toURLSearchParams({
    page: req.page,
    filter: req.filter,
  })
  const path = encodePath(
    'openmeter/customers/{customerId}/credits/transactions',
    { customerId: req.customerId },
  )
  return request(() =>
    http(client)
      .get(path, { ...options, searchParams })
      .json<ListCreditTransactionsResponse>(),
  )
}

export function listCustomerCharges(
  client: Client,
  req: ListCustomerChargesRequest,
  options?: RequestOptions,
): Promise<Result<ListCustomerChargesResponse>> {
  const searchParams = toURLSearchParams({
    page: req.page,
    sort: encodeSort(req.sort),
    filter: req.filter,
    expand: req.expand,
  })
  const path = encodePath('openmeter/customers/{customerId}/charges', {
    customerId: req.customerId,
  })
  return request(() =>
    http(client)
      .get(path, { ...options, searchParams })
      .json<ListCustomerChargesResponse>(),
  )
}

export function createCustomerCharges(
  client: Client,
  req: CreateCustomerChargesRequest,
  options?: RequestOptions,
): Promise<Result<CreateCustomerChargesResponse>> {
  const path = encodePath('openmeter/customers/{customerId}/charges', {
    customerId: req.customerId,
  })
  return request(() =>
    http(client)
      .post(path, { ...options, json: req.body })
      .json<CreateCustomerChargesResponse>(),
  )
}
