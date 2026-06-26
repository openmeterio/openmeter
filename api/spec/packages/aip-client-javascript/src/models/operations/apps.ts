import { z } from 'zod'
import * as schemas from '../schemas.js'
import type {
  AppPagePaginatedResponse,
  ListAppsParamsFilter,
  SortQueryInput,
} from '../types.js'

export interface ListAppsQuery {
  /** Determines which page of the collection to retrieve. */
  page?: { size?: number; number?: number }
  /** Sort apps returned in the response. Supported sort attributes are: - `id` - `created_at` (default) The `asc` suffix is optional as the default sort order is ascending. The `desc` suffix is used to specify a descending order. */
  sort?: SortQueryInput
  /** Filter apps returned in the response. To filter apps by name add the following query param: filter[name]=my-app */
  filter?: ListAppsParamsFilter
}

export type ListAppsRequest = ListAppsQuery
export type ListAppsResponse = AppPagePaginatedResponse

export type GetAppRequest = {
  appId: string
}
export type GetAppResponse = z.output<typeof schemas.getAppResponse>
