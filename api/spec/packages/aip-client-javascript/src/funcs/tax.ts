import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { toURLSearchParams, encodeSort } from '../lib/encodings.js'
import { toWire, fromWire, assertValid } from '../lib/wire.js'
import * as schemas from '../models/schemas.js'
import type {
  CreateTaxCodeRequest,
  CreateTaxCodeResponse,
  GetTaxCodeRequest,
  GetTaxCodeResponse,
  ListTaxCodesRequest,
  ListTaxCodesResponse,
  UpsertTaxCodeRequest,
  UpsertTaxCodeResponse,
  DeleteTaxCodeRequest,
  DeleteTaxCodeResponse,
} from '../models/operations/tax.js'

export function createTaxCode(
  client: Client,
  req: CreateTaxCodeRequest,
  options?: RequestOptions,
): Promise<Result<CreateTaxCodeResponse>> {
  return request(() => {
    const body = toWire(req, schemas.createTaxCodeBody)
    if (client._options.validate) {
      assertValid(schemas.createTaxCodeBodyWire, body)
    }
    return http(client)
      .post('openmeter/tax-codes', { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.createTaxCodeResponseWire, data)
        }
        return fromWire(data, schemas.createTaxCodeResponse)
      })
  })
}

export function getTaxCode(
  client: Client,
  req: GetTaxCodeRequest,
  options?: RequestOptions,
): Promise<Result<GetTaxCodeResponse>> {
  const path = `openmeter/tax-codes/${encodeURIComponent(String(req.taxCodeId))}`
  return request(() =>
    http(client)
      .get(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.getTaxCodeResponseWire, data)
        }
        return fromWire(data, schemas.getTaxCodeResponse)
      }),
  )
}

export function listTaxCodes(
  client: Client,
  req: ListTaxCodesRequest = {},
  options?: RequestOptions,
): Promise<Result<ListTaxCodesResponse>> {
  const searchParams = toURLSearchParams(
    toWire(
      {
        page: req.page,
        includeDeleted: req.includeDeleted,
      },
      schemas.listTaxCodesQueryParams,
    ),
  )
  return request(() =>
    http(client)
      .get('openmeter/tax-codes', { ...options, searchParams })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.listTaxCodesResponseWire, data)
        }
        return fromWire(data, schemas.listTaxCodesResponse)
      }),
  )
}

export function upsertTaxCode(
  client: Client,
  req: UpsertTaxCodeRequest,
  options?: RequestOptions,
): Promise<Result<UpsertTaxCodeResponse>> {
  const path = `openmeter/tax-codes/${encodeURIComponent(String(req.taxCodeId))}`
  return request(() => {
    const body = toWire(req.body, schemas.upsertTaxCodeBody)
    if (client._options.validate) {
      assertValid(schemas.upsertTaxCodeBodyWire, body)
    }
    return http(client)
      .put(path, { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.upsertTaxCodeResponseWire, data)
        }
        return fromWire(data, schemas.upsertTaxCodeResponse)
      })
  })
}

export function deleteTaxCode(
  client: Client,
  req: DeleteTaxCodeRequest,
  options?: RequestOptions,
): Promise<Result<DeleteTaxCodeResponse>> {
  const path = `openmeter/tax-codes/${encodeURIComponent(String(req.taxCodeId))}`
  return request(async () => {
    await http(client).delete(path, options)
  })
}
