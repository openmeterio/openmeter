import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { encodePath, toURLSearchParams, encodeSort } from '../lib/encodings.js'
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
  return request(() =>
    http(client)
      .post('openmeter/tax-codes', { ...options, json: req })
      .json<CreateTaxCodeResponse>(),
  )
}

export function getTaxCode(
  client: Client,
  req: GetTaxCodeRequest,
  options?: RequestOptions,
): Promise<Result<GetTaxCodeResponse>> {
  const path = encodePath('openmeter/tax-codes/{taxCodeId}', { taxCodeId: req.taxCodeId })
  return request(() =>
    http(client)
      .get(path, options)
      .json<GetTaxCodeResponse>(),
  )
}

export function listTaxCodes(
  client: Client,
  req: ListTaxCodesRequest = {},
  options?: RequestOptions,
): Promise<Result<ListTaxCodesResponse>> {
  const searchParams = toURLSearchParams({
    page: req.page,
    include_deleted: req.include_deleted,
  })
  return request(() =>
    http(client)
      .get('openmeter/tax-codes', { ...options, searchParams })
      .json<ListTaxCodesResponse>(),
  )
}

export function upsertTaxCode(
  client: Client,
  req: UpsertTaxCodeRequest,
  options?: RequestOptions,
): Promise<Result<UpsertTaxCodeResponse>> {
  const path = encodePath('openmeter/tax-codes/{taxCodeId}', { taxCodeId: req.taxCodeId })
  return request(() =>
    http(client)
      .put(path, { ...options, json: req.body })
      .json<UpsertTaxCodeResponse>(),
  )
}

export function deleteTaxCode(
  client: Client,
  req: DeleteTaxCodeRequest,
  options?: RequestOptions,
): Promise<Result<DeleteTaxCodeResponse>> {
  const path = encodePath('openmeter/tax-codes/{taxCodeId}', { taxCodeId: req.taxCodeId })
  return request(async () => {
    await http(client).delete(path, options)
  })
}
