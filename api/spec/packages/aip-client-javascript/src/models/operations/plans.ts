import { z } from 'zod'
import * as schemas from '../schemas.js'
import type {
  CreatePlanRequestInput,
  ListPlansParamsFilter,
  Plan,
  PlanPagePaginatedResponse,
  SortQueryInput,
  UpsertPlanRequestInput,
} from '../types.js'

export interface ListPlansQuery {
  /** Determines which page of the collection to retrieve. */
  page?: { size?: number; number?: number }
  /** Sort plans returned in the response. Supported sort attributes are: - `id` - `key` - `version` - `created_at` (default) - `updated_at` */
  sort?: SortQueryInput
  /** Filter plans returned in the response. */
  filter?: ListPlansParamsFilter
}

export type ListPlansRequest = ListPlansQuery
export type ListPlansResponse = PlanPagePaginatedResponse

export type CreatePlanRequest = CreatePlanRequestInput
export type CreatePlanResponse = Plan

export type UpdatePlanRequest = {
  planId: string
  body: UpsertPlanRequestInput
}
export type UpdatePlanResponse = Plan

export type GetPlanRequest = {
  planId: string
}
export type GetPlanResponse = Plan

export type DeletePlanRequest = {
  planId: string
}
export type DeletePlanResponse = void

export type ArchivePlanRequest = {
  planId: string
}
export type ArchivePlanResponse = Plan

export type PublishPlanRequest = {
  planId: string
}
export type PublishPlanResponse = Plan
