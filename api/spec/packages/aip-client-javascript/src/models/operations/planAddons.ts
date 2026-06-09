import { z } from 'zod'
import * as schemas from '../schemas.js'
import type {
  CreatePlanAddonRequest as CreatePlanAddonRequestBody,
  PlanAddon,
  PlanAddonPagePaginatedResponse,
  UpsertPlanAddonRequest,
} from '../types.js'

export interface ListPlanAddonsQuery {
  /** Determines which page of the collection to retrieve. */
  page?: { size?: number; number?: number }
}

export type ListPlanAddonsRequest = ListPlanAddonsQuery & { planId: string }
export type ListPlanAddonsResponse = PlanAddonPagePaginatedResponse

export type CreatePlanAddonRequest = {
  planId: string
  body: CreatePlanAddonRequestBody
}
export type CreatePlanAddonResponse = PlanAddon

export type GetPlanAddonRequest = {
  planId: string
  planAddonId: string
}
export type GetPlanAddonResponse = PlanAddon

export type UpdatePlanAddonRequest = {
  planId: string
  planAddonId: string
  body: UpsertPlanAddonRequest
}
export type UpdatePlanAddonResponse = PlanAddon

export type DeletePlanAddonRequest = {
  planId: string
  planAddonId: string
}
export type DeletePlanAddonResponse = void
