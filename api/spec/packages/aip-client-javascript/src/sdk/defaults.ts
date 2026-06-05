import { type Client } from '../core.js'
import { unwrap, type RequestOptions } from '../lib/types.js'
import {
  getOrganizationDefaultTaxCodes,
  updateOrganizationDefaultTaxCodes,
} from '../funcs/defaults.js'
import type {
  GetOrganizationDefaultTaxCodesRequest,
  GetOrganizationDefaultTaxCodesResponse,
  UpdateOrganizationDefaultTaxCodesRequest,
  UpdateOrganizationDefaultTaxCodesResponse,
} from '../models/operations/defaults.js'

export class Defaults {
  constructor(private readonly _client: Client) {}

  async getOrganizationTaxCodes(
    request: GetOrganizationDefaultTaxCodesRequest,
    options?: RequestOptions,
  ): Promise<GetOrganizationDefaultTaxCodesResponse> {
    return unwrap(
      await getOrganizationDefaultTaxCodes(this._client, request, options),
    )
  }

  async updateOrganizationTaxCodes(
    request: UpdateOrganizationDefaultTaxCodesRequest,
    options?: RequestOptions,
  ): Promise<UpdateOrganizationDefaultTaxCodesResponse> {
    return unwrap(
      await updateOrganizationDefaultTaxCodes(this._client, request, options),
    )
  }
}
