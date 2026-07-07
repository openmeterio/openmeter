import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { fromWire, assertValid } from '../lib/wire.js'
import * as schemas from '../models/schemas.js'
import type {
  ListCustomerEntitlementAccessRequest,
  ListCustomerEntitlementAccessResponse,
} from '../models/operations/entitlements.js'

export function listCustomerEntitlementAccess(
  client: Client,
  req: ListCustomerEntitlementAccessRequest,
  options?: RequestOptions,
): Promise<Result<ListCustomerEntitlementAccessResponse>> {
  return request(() => {
    const path = `openmeter/customers/${(() => {
      if (req.customerId === undefined) {
        throw new Error('missing path parameter: customerId')
      }
      return encodeURIComponent(String(req.customerId))
    })()}/entitlement-access`
    return http(client)
      .get(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.listCustomerEntitlementAccessResponseWire, data)
        }
        return fromWire(data, schemas.listCustomerEntitlementAccessResponse)
      })
  })
}
