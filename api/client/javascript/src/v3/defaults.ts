import type { Client } from 'openapi-fetch'
import type { RequestOptions } from '../client/common.js'
import { transformResponse } from '../client/utils.js'
import type {
  paths,
  UpdateOrganizationDefaultTaxCodesRequest,
} from './schemas.js'

/**
 * Defaults (v3)
 *
 * Thin wrapper over the v3 organization defaults endpoints. Bodies use the v3
 * wire shape verbatim (snake_case); no field renaming (Option A).
 */
export class Defaults {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Get organization default tax codes
   */
  public async getOrganizationTaxCodes(options?: RequestOptions) {
    const resp = await this.client.GET('/openmeter/defaults/tax-codes', {
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Update organization default tax codes
   */
  public async updateOrganizationTaxCodes(
    body: UpdateOrganizationDefaultTaxCodesRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT('/openmeter/defaults/tax-codes', {
      body,
      ...options,
    })

    return transformResponse(resp)
  }
}
