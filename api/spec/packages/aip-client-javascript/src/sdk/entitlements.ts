import { type Client } from '../core.js'
import { unwrap, type RequestOptions } from '../lib/types.js'
import {
  listCustomerEntitlementAccess,
} from '../funcs/entitlements.js'
import type {
  ListCustomerEntitlementAccessRequest,
  ListCustomerEntitlementAccessResponse,
} from '../models/operations/entitlements.js'

export class Entitlements {
  constructor(private readonly _client: Client) {}

  async listCustomerAccess(
    request: ListCustomerEntitlementAccessRequest,
    options?: RequestOptions,
  ): Promise<ListCustomerEntitlementAccessResponse> {
    return unwrap(await listCustomerEntitlementAccess(this._client, request, options))
  }
}
