import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { fromWire, assertValid } from '../lib/wire.js'
import * as schemas from '../models/schemas.js'
import type {
  GetInvoiceRequest,
  GetInvoiceResponse,
} from '../models/operations/invoices.js'

export function getInvoice(
  client: Client,
  req: GetInvoiceRequest,
  options?: RequestOptions,
): Promise<Result<GetInvoiceResponse>> {
  const path = `openmeter/billing/invoices/${(() => {
    if (req.invoiceId === undefined) {
      throw new Error('missing path parameter: invoiceId')
    }
    return encodeURIComponent(String(req.invoiceId))
  })()}`
  return request(() =>
    http(client)
      .get(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.getInvoiceResponseWire, data)
        }
        return fromWire(data, schemas.getInvoiceResponse)
      }),
  )
}
