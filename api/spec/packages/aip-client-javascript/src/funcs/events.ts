import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { encodePath, toURLSearchParams, encodeSort } from '../lib/encodings.js'
import type {
  ListMeteringEventsRequest,
  ListMeteringEventsResponse,
  IngestMeteringEventsRequest,
  IngestMeteringEventsResponse,
  ListEventSubjectsRequest,
  ListEventSubjectsResponse,
} from '../models/operations/events.js'

export function listMeteringEvents(
  client: Client,
  req: ListMeteringEventsRequest = {},
  options?: RequestOptions,
): Promise<Result<ListMeteringEventsResponse>> {
  const searchParams = toURLSearchParams({
    page: req.page,
    filter: req.filter,
    sort: encodeSort(req.sort),
  })
  return request(() =>
    http(client)
      .get('openmeter/events', { ...options, searchParams })
      .json<ListMeteringEventsResponse>(),
  )
}

export function ingestMeteringEvents(
  client: Client,
  req: IngestMeteringEventsRequest,
  options?: RequestOptions,
): Promise<Result<IngestMeteringEventsResponse>> {
  return request(async () => {
    await http(client).post('openmeter/events', { ...options, json: req })
  })
}

export function listEventSubjects(
  client: Client,
  req: ListEventSubjectsRequest = {},
  options?: RequestOptions,
): Promise<Result<ListEventSubjectsResponse>> {
  const searchParams = toURLSearchParams({
    page: req.page,
    filter: req.filter,
  })
  return request(() =>
    http(client)
      .get('openmeter/events/subjects', { ...options, searchParams })
      .json<ListEventSubjectsResponse>(),
  )
}
