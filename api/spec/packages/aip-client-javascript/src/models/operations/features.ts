import { z } from 'zod'
import * as schemas from '../schemas.js'
import type {
  CreateFeatureRequest as CreateFeatureRequestBody,
  Feature,
  FeatureCostQueryResult,
  FeaturePagePaginatedResponse,
  ListFeatureParamsFilter,
  MeterQueryRequestInput,
  SortQueryInput,
  UpdateFeatureRequest as UpdateFeatureRequestBody,
} from '../types.js'

export interface ListFeaturesQuery {
  /** Determines which page of the collection to retrieve. */
  page?: { size?: number; number?: number }
  /**
   * Sort features returned in the response. Supported sort attributes are:
   *
   * - `key`
   * - `name`
   * - `created_at` (default)
   * - `updated_at`
   *
   * The `asc` suffix is optional as the default sort order is ascending. The `desc`
   * suffix is used to specify a descending order.
   */
  sort?: SortQueryInput
  /**
   * Filter features returned in the response.
   *
   * To filter features by meter_id add the following query param:
   * filter[meter_id][oeq]=<id>
   */
  filter?: ListFeatureParamsFilter
}

export type ListFeaturesRequest = ListFeaturesQuery
export type ListFeaturesResponse = FeaturePagePaginatedResponse

export type CreateFeatureRequest = CreateFeatureRequestBody
export type CreateFeatureResponse = Feature

export type GetFeatureRequest = {
  featureId: string
}
export type GetFeatureResponse = Feature

export type UpdateFeatureRequest = {
  featureId: string
  body: UpdateFeatureRequestBody
}
export type UpdateFeatureResponse = Feature

export type DeleteFeatureRequest = {
  featureId: string
}
export type DeleteFeatureResponse = void

export type QueryFeatureCostRequest = {
  featureId: string
  body: MeterQueryRequestInput
}
export type QueryFeatureCostResponse = FeatureCostQueryResult
