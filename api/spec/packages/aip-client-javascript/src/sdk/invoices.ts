import { type Client } from '../core.js'
import { unwrap, type RequestOptions } from '../lib/types.js'
import { listInvoices, getInvoice } from '../funcs/invoices.js'
import type {
  ListInvoicesRequest,
  ListInvoicesResponse,
  GetInvoiceRequest,
  GetInvoiceResponse,
} from '../models/operations/invoices.js'

export class Invoices {
  constructor(private readonly _client: Client) {}

  async list(
    request?: ListInvoicesRequest,
    options?: RequestOptions,
  ): Promise<ListInvoicesResponse> {
    return unwrap(await listInvoices(this._client, request, options))
  }

  async get(
    request: GetInvoiceRequest,
    options?: RequestOptions,
  ): Promise<GetInvoiceResponse> {
    return unwrap(await getInvoice(this._client, request, options))
  }
}
