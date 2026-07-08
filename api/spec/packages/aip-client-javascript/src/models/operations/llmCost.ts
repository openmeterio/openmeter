import { z } from 'zod'
import * as schemas from '../schemas.js'
import type { AcceptDateStrings } from '../../lib/wire.js'
import type {
  ListLlmCostPricesParamsFilter,
  LlmCostOverrideCreate,
  LlmCostPrice,
  PricePagePaginatedResponse,
  SortQueryInput,
} from '../types.js'

export interface ListLlmCostPricesQuery {
  /** Filter prices. */
  filter?: ListLlmCostPricesParamsFilter
  /**
   * Sort prices returned in the response. Supported sort attributes are:
   *
   * - `id`
   * - `provider.id`
   * - `model.id` (default)
   * - `effective_from`
   * - `effective_to`
   *
   * The `asc` suffix is optional as the default sort order is ascending. The `desc`
   * suffix is used to specify a descending order.
   */
  sort?: SortQueryInput
  /** Determines which page of the collection to retrieve. */
  page?: { size?: number; number?: number }
}

export type ListLlmCostPricesRequest = AcceptDateStrings<ListLlmCostPricesQuery>
export type ListLlmCostPricesResponse = PricePagePaginatedResponse

export type GetLlmCostPriceRequest = {
  priceId: string
}
export type GetLlmCostPriceResponse = LlmCostPrice

export interface ListLlmCostOverridesQuery {
  filter?: ListLlmCostPricesParamsFilter
  /** Determines which page of the collection to retrieve. */
  page?: { size?: number; number?: number }
}

export type ListLlmCostOverridesRequest =
  AcceptDateStrings<ListLlmCostOverridesQuery>
export type ListLlmCostOverridesResponse = PricePagePaginatedResponse

export type CreateLlmCostOverrideRequest =
  AcceptDateStrings<LlmCostOverrideCreate>
export type CreateLlmCostOverrideResponse = LlmCostPrice

export type DeleteLlmCostOverrideRequest = {
  priceId: string
}
export type DeleteLlmCostOverrideResponse = void
