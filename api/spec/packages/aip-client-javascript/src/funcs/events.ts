import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { toURLSearchParams, encodeSort } from '../lib/encodings.js'
import { toWire, fromWire, assertValid, toSnakeCase } from '../lib/wire.js'
import * as schemas from '../models/schemas.js'
import type {
  ListMeteringEventsRequest,
  ListMeteringEventsResponse,
  IngestMeteringEventsRequest,
  IngestMeteringEventsResponse,
} from '../models/operations/events.js'

export function listMeteringEvents(
  client: Client,
  req: ListMeteringEventsRequest = {},
  options?: RequestOptions,
): Promise<Result<ListMeteringEventsResponse>> {
  const searchParams = toURLSearchParams(
    toWire(
      {
        page: req.page,
        filter: req.filter,
        sort: encodeSort(req.sort, toSnakeCase),
      },
      schemas.listMeteringEventsQueryParams,
    ),
  )
  return request(() =>
    http(client)
      .get('openmeter/events', { ...options, searchParams })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.listMeteringEventsResponseWire, data)
        }
        return fromWire(data, schemas.listMeteringEventsResponse)
      }),
  )
}

export function ingestMeteringEvents(
  client: Client,
  req: IngestMeteringEventsRequest,
  options?: RequestOptions,
): Promise<Result<IngestMeteringEventsResponse>> {
  return request(async () => {
    const body = toWire(req, schemas.ingestMeteringEventsBody)
    if (client._options.validate) {
      assertValid(schemas.ingestMeteringEventsBodyWire, body)
    }
    await http(client).post('openmeter/events', { ...options, json: body })
  })
}
