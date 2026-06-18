import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { encodePath, toURLSearchParams, encodeSort } from '../lib/encodings.js'
import type {
  GetBillingInvoiceRequest,
  GetBillingInvoiceResponse,
} from '../models/operations/invoices.js'

export function getBillingInvoice(
  client: Client,
  req: GetBillingInvoiceRequest,
  options?: RequestOptions,
): Promise<Result<GetBillingInvoiceResponse>> {
  const path = encodePath('openmeter/billing/invoices/{invoiceId}', {
    invoiceId: req.invoiceId,
  })
  return request(() =>
    http(client).get(path, options).json<GetBillingInvoiceResponse>(),
  )
}
