import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { encodePath, toURLSearchParams, encodeSort } from '../lib/encodings.js'
import type {
  ListCustomerEntitlementAccessRequest,
  ListCustomerEntitlementAccessResponse,
} from '../models/operations/entitlements.js'

export function listCustomerEntitlementAccess(
  client: Client,
  req: ListCustomerEntitlementAccessRequest,
  options?: RequestOptions,
): Promise<Result<ListCustomerEntitlementAccessResponse>> {
  const path = encodePath('openmeter/customers/{customerId}/entitlement-access', { customerId: req.customerId })
  return request(() =>
    http(client)
      .get(path, options)
      .json<ListCustomerEntitlementAccessResponse>(),
  )
}
