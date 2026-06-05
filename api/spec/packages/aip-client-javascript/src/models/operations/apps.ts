import { z } from 'zod'
import * as schemas from '../schemas.js'
import type { AppPagePaginatedResponse } from '../types.js'

export interface ListAppsQuery {
  /** Determines which page of the collection to retrieve. */
  page?: { size?: number; number?: number }
}

export type ListAppsRequest = ListAppsQuery
export type ListAppsResponse = AppPagePaginatedResponse

export type GetAppRequest = {
  appId: string
}
export type GetAppResponse = z.output<typeof schemas.getAppResponse>
