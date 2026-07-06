import { type Client } from '../core.js'
import { unwrap, type RequestOptions } from '../lib/types.js'
import { listInvoices, getInvoice, updateInvoice } from '../funcs/invoices.js'
import type {
  ListInvoicesRequest,
  ListInvoicesResponse,
  GetInvoiceRequest,
  GetInvoiceResponse,
  UpdateInvoiceRequest,
  UpdateInvoiceResponse,
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

  async update(
    request: UpdateInvoiceRequest,
    options?: RequestOptions,
  ): Promise<UpdateInvoiceResponse> {
    return unwrap(await updateInvoice(this._client, request, options))
  }
}
