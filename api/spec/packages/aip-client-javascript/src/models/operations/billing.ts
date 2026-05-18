import { z } from 'zod'
import * as schemas from '../schemas.js'
import type {
  CreateBillingProfileRequestInput,
  ListBillingProfilesParamsFilter,
  Profile,
  ProfilePagePaginatedResponse,
  SortQueryInput,
  UpsertBillingProfileRequestInput,
} from '../types.js'

export interface ListBillingProfilesQuery {
  /** Determines which page of the collection to retrieve. */
  page?: { size?: number; number?: number }
  /** Sort billing profiles returned in the response. Supported sort attributes are: - `id` - `name` - `createdAt` (default) - `updatedAt` The `asc` suffix is optional as the default sort order is ascending. The `desc` suffix is used to specify a descending order. */
  sort?: SortQueryInput
  /** Filter billing profiles returned in the response. To filter billing profiles by name add the following query param: filter[name]=my-profile */
  filter?: ListBillingProfilesParamsFilter
}

export type ListBillingProfilesRequest = ListBillingProfilesQuery
export type ListBillingProfilesResponse = ProfilePagePaginatedResponse

export type CreateBillingProfileRequest = CreateBillingProfileRequestInput
export type CreateBillingProfileResponse = Profile

export type GetBillingProfileRequest = {
  id: string
}
export type GetBillingProfileResponse = Profile

export type UpdateBillingProfileRequest = {
  id: string
  body: UpsertBillingProfileRequestInput
}
export type UpdateBillingProfileResponse = Profile

export type DeleteBillingProfileRequest = {
  id: string
}
export type DeleteBillingProfileResponse = void
