import { z } from 'zod'
import * as schemas from '../schemas.js'
import type {
  CreateMeterRequest as CreateMeterRequestBody,
  ListMetersParamsFilter,
  Meter,
  MeterPagePaginatedResponse,
  MeterQueryRequestInput,
  MeterQueryResult,
  SortQueryInput,
  UpdateMeterRequest as UpdateMeterRequestBody,
} from '../types.js'

export type CreateMeterRequest = CreateMeterRequestBody
export type CreateMeterResponse = Meter

export type GetMeterRequest = {
  meterId: string
}
export type GetMeterResponse = Meter

export interface ListMetersQuery {
  /** Determines which page of the collection to retrieve. */
  page?: { size?: number; number?: number }
  /** Sort meters returned in the response. Supported sort attributes are: - `key` - `name` - `aggregation` - `createdAt` (default) - `updatedAt` The `asc` suffix is optional as the default sort order is ascending. The `desc` suffix is used to specify a descending order. */
  sort?: SortQueryInput
  /** Filter meters returned in the response. To filter meters by key add the following query param: filter[key]=my-meter-key */
  filter?: ListMetersParamsFilter
}

export type ListMetersRequest = ListMetersQuery
export type ListMetersResponse = MeterPagePaginatedResponse

export type UpdateMeterRequest = {
  meterId: string
  body: UpdateMeterRequestBody
}
export type UpdateMeterResponse = Meter

export type DeleteMeterRequest = {
  meterId: string
}
export type DeleteMeterResponse = void

export type QueryMeterRequest = {
  meterId: string
  body: MeterQueryRequestInput
}
export type QueryMeterResponse = MeterQueryResult

export type QueryMeterCsvRequest = {
  meterId: string
  body: MeterQueryRequestInput
}
export type QueryMeterCsvResponse = string
