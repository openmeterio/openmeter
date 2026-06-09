import { z } from 'zod'
import * as schemas from '../schemas.js'
import type {
  CursorPaginationQueryPage,
  EventInput,
  IngestedEventPaginatedResponse,
  ListEventsParamsFilter,
  ListSubjectsParamsFilter,
  SortQueryInput,
  SubjectPaginatedResponse,
} from '../types.js'

export interface ListMeteringEventsQuery {
  page?: CursorPaginationQueryPage
  /** Filter events returned in the response. To filter events by subject add the following query param: filter[subject][eq]=customer-1 */
  filter?: ListEventsParamsFilter
  /** Sort events returned in the response. Supported sort attributes are: - `time` (default) - `ingested_at` - `stored_at` When omitted, events are sorted by `time desc` (most recent first). When a sort field is provided without a suffix, it sorts descending. Append the `asc` suffix to sort ascending, or the `desc` suffix to sort descending. */
  sort?: SortQueryInput
}

export type ListMeteringEventsRequest = ListMeteringEventsQuery
export type ListMeteringEventsResponse = IngestedEventPaginatedResponse

export type IngestMeteringEventsRequest = EventInput | EventInput[]
export type IngestMeteringEventsResponse = void

export interface ListEventSubjectsQuery {
  page?: CursorPaginationQueryPage
  /** Filter subjects returned in the response. To filter subjects by key add the following query param: filter[key][contains]=customer */
  filter?: ListSubjectsParamsFilter
}

export type ListEventSubjectsRequest = ListEventSubjectsQuery
export type ListEventSubjectsResponse = SubjectPaginatedResponse
