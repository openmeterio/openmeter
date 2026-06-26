import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { encodePath, toURLSearchParams, encodeSort } from '../lib/encodings.js'
import type {
  GetInvoiceRequest,
  GetInvoiceResponse,
} from '../models/operations/invoices.js'

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
