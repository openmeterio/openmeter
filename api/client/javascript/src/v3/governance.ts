import type { Client } from 'openapi-fetch'
import type { RequestOptions } from '../client/common.js'
import { transformResponse } from '../client/utils.js'
import type { GovernanceQueryRequest, paths } from './schemas.js'

/**
 * Governance (v3)
 *
 * Thin wrapper over the v3 governance endpoints. Bodies use the v3 wire shape
 * verbatim (snake_case); no field renaming (Option A).
 */
export class Governance {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Query feature access for a list of customers
   */
  public async queryAccess(
    body: GovernanceQueryRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST('/openmeter/governance/query', {
      body,
      ...options,
    })

    return transformResponse(resp)
  }
}
