import { type Client } from '../core.js'
import { unwrap, type RequestOptions } from '../lib/types.js'
import {
  listLlmCostPrices,
  getLlmCostPrice,
  listLlmCostOverrides,
  createLlmCostOverride,
  deleteLlmCostOverride,
} from '../funcs/llmCost.js'
import type {
  ListLlmCostPricesRequest,
  ListLlmCostPricesResponse,
  GetLlmCostPriceRequest,
  GetLlmCostPriceResponse,
  ListLlmCostOverridesRequest,
  ListLlmCostOverridesResponse,
  CreateLlmCostOverrideRequest,
  CreateLlmCostOverrideResponse,
  DeleteLlmCostOverrideRequest,
  DeleteLlmCostOverrideResponse,
} from '../models/operations/llmCost.js'

export class LLMCost {
  constructor(private readonly _client: Client) {}

  async listPrices(
    request?: ListLlmCostPricesRequest,
    options?: RequestOptions,
  ): Promise<ListLlmCostPricesResponse> {
    return unwrap(await listLlmCostPrices(this._client, request, options))
  }

  async getPrice(
    request: GetLlmCostPriceRequest,
    options?: RequestOptions,
  ): Promise<GetLlmCostPriceResponse> {
    return unwrap(await getLlmCostPrice(this._client, request, options))
  }

  async listOverrides(
    request?: ListLlmCostOverridesRequest,
    options?: RequestOptions,
  ): Promise<ListLlmCostOverridesResponse> {
    return unwrap(await listLlmCostOverrides(this._client, request, options))
  }

  async createOverride(
    request: CreateLlmCostOverrideRequest,
    options?: RequestOptions,
  ): Promise<CreateLlmCostOverrideResponse> {
    return unwrap(await createLlmCostOverride(this._client, request, options))
  }

  async deleteOverride(
    request: DeleteLlmCostOverrideRequest,
    options?: RequestOptions,
  ): Promise<DeleteLlmCostOverrideResponse> {
    return unwrap(await deleteLlmCostOverride(this._client, request, options))
  }
}
