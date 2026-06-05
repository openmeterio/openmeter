import { z } from 'zod'
import * as schemas from '../schemas.js'
import type {
  CreateTaxCodeRequest as CreateTaxCodeRequestBody,
  TaxCode,
  TaxCodePagePaginatedResponse,
  UpsertTaxCodeRequest as UpsertTaxCodeRequestBody,
} from '../types.js'

export type CreateTaxCodeRequest = CreateTaxCodeRequestBody
export type CreateTaxCodeResponse = TaxCode

export type GetTaxCodeRequest = {
  taxCodeId: string
}
export type GetTaxCodeResponse = TaxCode

export interface ListTaxCodesQuery {
  /** Determines which page of the collection to retrieve. */
  page?: { size?: number; number?: number }
  /** Include deleted tax codes in the response. */
  include_deleted?: boolean
}

export type ListTaxCodesRequest = ListTaxCodesQuery
export type ListTaxCodesResponse = TaxCodePagePaginatedResponse

export type UpsertTaxCodeRequest = {
  taxCodeId: string
  body: UpsertTaxCodeRequestBody
}
export type UpsertTaxCodeResponse = TaxCode

export type DeleteTaxCodeRequest = {
  taxCodeId: string
}
export type DeleteTaxCodeResponse = void
