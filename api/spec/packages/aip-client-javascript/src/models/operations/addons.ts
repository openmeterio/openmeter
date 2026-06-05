import { z } from 'zod'
import * as schemas from '../schemas.js'
import type {
  Addon,
  AddonPagePaginatedResponse,
  CreateAddonRequestInput,
  ListAddonsParamsFilter,
  SortQueryInput,
  UpsertAddonRequestInput,
} from '../types.js'

export interface ListAddonsQuery {
  /** Determines which page of the collection to retrieve. */
  page?: { size?: number; number?: number }
  /** Sort add-ons returned in the response. Supported sort attributes are: - `id` - `key` - `name` - `created_at` (default) - `updated_at` The `asc` suffix is optional as the default sort order is ascending. The `desc` suffix is used to specify a descending order. */
  sort?: SortQueryInput
  /** Filter add-ons returned in the response. */
  filter?: ListAddonsParamsFilter
}

export type ListAddonsRequest = ListAddonsQuery
export type ListAddonsResponse = AddonPagePaginatedResponse

export type CreateAddonRequest = CreateAddonRequestInput
export type CreateAddonResponse = Addon

export type UpdateAddonRequest = {
  addonId: string
  body: UpsertAddonRequestInput
}
export type UpdateAddonResponse = Addon

export type GetAddonRequest = {
  addonId: string
}
export type GetAddonResponse = Addon

export type DeleteAddonRequest = {
  addonId: string
}
export type DeleteAddonResponse = void

export type ArchiveAddonRequest = {
  addonId: string
}
export type ArchiveAddonResponse = Addon

export type PublishAddonRequest = {
  addonId: string
}
export type PublishAddonResponse = Addon
