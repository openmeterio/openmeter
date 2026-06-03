import type { Client } from 'openapi-fetch'
import type { RequestOptions } from '../client/common.js'
import { transformResponse } from '../client/utils.js'
import type { LlmCostOverrideCreate, operations, paths } from './schemas.js'

/**
 * LLM cost (v3)
 *
 * Thin wrapper over the v3 LLM cost endpoints. Bodies use the v3 wire shape
 * verbatim (snake_case); no field renaming (Option A).
 */
export class LlmCost {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Create an LLM cost override
   */
  public async createOverride(
    body: LlmCostOverrideCreate,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST('/openmeter/llm-cost/overrides', {
      body,
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Delete an LLM cost override
   */
  public async deleteOverride(priceId: string, options?: RequestOptions) {
    const resp = await this.client.DELETE(
      '/openmeter/llm-cost/overrides/{priceId}',
      {
        params: { path: { priceId } },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * List LLM cost overrides
   */
  public async listOverrides(
    params?: operations['list-llm-cost-overrides']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/openmeter/llm-cost/overrides', {
      params: { query: params },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get an LLM cost price by ID
   */
  public async getPrice(priceId: string, options?: RequestOptions) {
    const resp = await this.client.GET('/openmeter/llm-cost/prices/{priceId}', {
      params: { path: { priceId } },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List LLM cost prices
   */
  public async listPrices(
    params?: operations['list-llm-cost-prices']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/openmeter/llm-cost/prices', {
      params: { query: params },
      ...options,
    })

    return transformResponse(resp)
  }
}
