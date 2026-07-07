import { z } from 'zod'
import * as schemas from '../schemas.js'
import type {
  InvoicePagePaginatedResponse,
  ListInvoicesParamsFilter,
  SortQueryInput,
  UpdateInvoiceStandardRequestInput,
} from '../types.js'

export interface ListInvoicesQuery {
  /** Determines which page of the collection to retrieve. */
  page?: { size?: number; number?: number }
  /**
   * Sort invoices returned in the response. Supported sort attributes:
   *
   * - `issued_at`
   * - `created_at` (default)
   * - `service_period_start`
   *
   * The `asc` suffix is optional as the default sort order is ascending. The `desc`
   * suffix is used to specify a descending order.
   */
  sort?: SortQueryInput
  /**
   * Filter invoices returned in the response.
   *
   * Examples:
   *
   * - `filter[status][oeq]=draft,issued`
   * - `filter[customer_id]=01KPDB8K...`
   * - `filter[issued_at][gte]=2024-01-01T00:00:00Z`
   */
  filter?: ListInvoicesParamsFilter
}

export type ListInvoicesRequest = ListInvoicesQuery
export type ListInvoicesResponse = InvoicePagePaginatedResponse

export type GetInvoiceRequest = {
  invoiceId: string
}
export type GetInvoiceResponse = z.output<typeof schemas.getInvoiceResponse>

export type UpdateInvoiceRequest = {
  invoiceId: string
  body: UpdateInvoiceStandardRequestInput
}
export type UpdateInvoiceResponse = z.output<
  typeof schemas.updateInvoiceResponse
>

export type DeleteInvoiceRequest = {
  invoiceId: string
}
export type DeleteInvoiceResponse = void
