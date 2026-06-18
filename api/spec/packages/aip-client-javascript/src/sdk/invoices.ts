import { type Client } from '../core.js'
import { unwrap, type RequestOptions } from '../lib/types.js'
import { getBillingInvoice } from '../funcs/invoices.js'
import type {
  GetBillingInvoiceRequest,
  GetBillingInvoiceResponse,
} from '../models/operations/invoices.js'

export class Invoices {
  constructor(private readonly _client: Client) {}

  async getBilling(
    request: GetBillingInvoiceRequest,
    options?: RequestOptions,
  ): Promise<GetBillingInvoiceResponse> {
    return unwrap(await getBillingInvoice(this._client, request, options))
  }
}
