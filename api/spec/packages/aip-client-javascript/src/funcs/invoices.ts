import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { encodePath, toURLSearchParams, encodeSort } from '../lib/encodings.js'
import type {
  ListInvoicesRequest,
  ListInvoicesResponse,
  GetInvoiceRequest,
  GetInvoiceResponse,
  UpdateInvoiceRequest,
  UpdateInvoiceResponse,
} from '../models/operations/invoices.js'

export function listInvoices(
  client: Client,
  req: ListInvoicesRequest = {},
  options?: RequestOptions,
): Promise<Result<ListInvoicesResponse>> {
  const searchParams = toURLSearchParams({
    page: req.page,
    sort: encodeSort(req.sort),
    filter: req.filter,
  })
  return request(() =>
    http(client)
      .get('openmeter/billing/invoices', { ...options, searchParams })
      .json<ListInvoicesResponse>(),
  )
}

export function getInvoice(
  client: Client,
  req: GetInvoiceRequest,
  options?: RequestOptions,
): Promise<Result<GetInvoiceResponse>> {
  const path = encodePath('openmeter/billing/invoices/{invoiceId}', {
    invoiceId: req.invoiceId,
  })
  return request(() =>
    http(client).get(path, options).json<GetInvoiceResponse>(),
  )
}

export function updateInvoice(
  client: Client,
  req: UpdateInvoiceRequest,
  options?: RequestOptions,
): Promise<Result<UpdateInvoiceResponse>> {
  const path = encodePath('openmeter/billing/invoices/{invoiceId}', {
    invoiceId: req.invoiceId,
  })
  return request(() =>
    http(client)
      .put(path, { ...options, json: req.body })
      .json<UpdateInvoiceResponse>(),
  )
}
