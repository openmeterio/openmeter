import { z } from 'zod'
import * as schemas from '../schemas.js'
import type { CreateBillingProfileRequestInput, Profile, ProfilePagePaginatedResponse, UpsertBillingProfileRequestInput } from '../types.js'

export interface ListBillingProfilesQuery {
  /** Determines which page of the collection to retrieve. */
  page?: { size?: number; number?: number }
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
