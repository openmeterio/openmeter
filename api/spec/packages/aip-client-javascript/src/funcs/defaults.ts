import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { encodePath, toURLSearchParams, encodeSort } from '../lib/encodings.js'
import type {
  GetOrganizationDefaultTaxCodesRequest,
  GetOrganizationDefaultTaxCodesResponse,
  UpdateOrganizationDefaultTaxCodesRequest,
  UpdateOrganizationDefaultTaxCodesResponse,
} from '../models/operations/defaults.js'

export function getOrganizationDefaultTaxCodes(
  client: Client,
  req: GetOrganizationDefaultTaxCodesRequest,
  options?: RequestOptions,
): Promise<Result<GetOrganizationDefaultTaxCodesResponse>> {
  return request(() =>
    http(client)
      .get('openmeter/defaults/tax-codes', options)
      .json<GetOrganizationDefaultTaxCodesResponse>(),
  )
}

export function updateOrganizationDefaultTaxCodes(
  client: Client,
  req: UpdateOrganizationDefaultTaxCodesRequest,
  options?: RequestOptions,
): Promise<Result<UpdateOrganizationDefaultTaxCodesResponse>> {
  return request(() =>
    http(client)
      .put('openmeter/defaults/tax-codes', { ...options, json: req })
      .json<UpdateOrganizationDefaultTaxCodesResponse>(),
  )
}
