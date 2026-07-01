import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { toWire, fromWire, assertValid } from '../lib/wire.js'
import * as schemas from '../models/schemas.js'
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
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.getOrganizationDefaultTaxCodesResponseWire, data)
        }
        return fromWire(data, schemas.getOrganizationDefaultTaxCodesResponse)
      }),
  )
}

export function updateOrganizationDefaultTaxCodes(
  client: Client,
  req: UpdateOrganizationDefaultTaxCodesRequest,
  options?: RequestOptions,
): Promise<Result<UpdateOrganizationDefaultTaxCodesResponse>> {
  return request(() => {
    const body = toWire(req, schemas.updateOrganizationDefaultTaxCodesBody)
    if (client._options.validate) {
      assertValid(schemas.updateOrganizationDefaultTaxCodesBodyWire, body)
    }
    return http(client)
      .put('openmeter/defaults/tax-codes', { ...options, json: body })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(
            schemas.updateOrganizationDefaultTaxCodesResponseWire,
            data,
          )
        }
        return fromWire(data, schemas.updateOrganizationDefaultTaxCodesResponse)
      })
  })
}
