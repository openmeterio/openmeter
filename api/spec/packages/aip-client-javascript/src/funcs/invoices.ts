import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { toURLSearchParams, encodeSort } from '../lib/encodings.js'
import { toWire, fromWire, assertValid, toSnakeCase } from '../lib/wire.js'
import * as schemas from '../models/schemas.js'
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
  return request(() => {
    const query = toWire(
      {
        page: req.page,
        sort: encodeSort(req.sort, toSnakeCase),
        filter: req.filter,
      },
      schemas.listInvoicesQueryParams,
    )
    if (client._options.validate) {
      assertValid(schemas.listInvoicesQueryParamsWire, query)
    }
    const searchParams = toURLSearchParams(query)
    return http(client)
      .get('openmeter/billing/invoices', { ...options, searchParams })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.listInvoicesResponseWire, data)
        }
        return fromWire(data, schemas.listInvoicesResponse)
      })
  })
}

export function getInvoice(
  client: Client,
  req: GetInvoiceRequest,
  options?: RequestOptions,
): Promise<Result<GetInvoiceResponse>> {
  return request(() => {
    const path = `openmeter/billing/invoices/${(() => {
      if (req.invoiceId === undefined) {
        throw new Error('missing path parameter: invoiceId')
      }
      return encodeURIComponent(String(req.invoiceId))
    })()}`
    return http(client)
      .get(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.getInvoiceResponseWire, data)
        }
        return fromWire(data, schemas.getInvoiceResponse)
      })
  })
}

export function updateInvoice(
  client: Client,
  req: UpdateInvoiceRequest,
  options?: RequestOptions,
): Promise<Result<UpdateInvoiceResponse>> {
  return request(() => {
    const path = `openmeter/billing/invoices/${(() => {
      if (req.invoiceId === undefined) {
        throw new Error('missing path parameter: invoiceId')
      }
      return encodeURIComponent(String(req.invoiceId))
    })()}`
    const body = toWire(req.body, schemas.updateInvoiceBody)
    if (client._options.validate) {
      assertValid(schemas.updateInvoiceBodyWire, body)
    }
    return http(client)
      .put(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.updateInvoiceResponseWire, data)
        }
        return fromWire(data, schemas.updateInvoiceResponse)
      })
  })
}
