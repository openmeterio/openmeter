import { type Client } from '../core.js'
import { unwrap, type RequestOptions } from '../lib/types.js'
import { getInvoice } from '../funcs/invoices.js'
import type {
  GetInvoiceRequest,
  GetInvoiceResponse,
} from '../models/operations/invoices.js'

export class Invoices {
  constructor(private readonly _client: Client) {}

  async get(
    request: GetInvoiceRequest,
    options?: RequestOptions,
  ): Promise<GetInvoiceResponse> {
    return unwrap(await getInvoice(this._client, request, options))
  }
}
