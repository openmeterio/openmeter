import { type Client } from '../core.js'
import { unwrap, type RequestOptions } from '../lib/types.js'
import { listMeteringEvents, ingestMeteringEvents } from '../funcs/events.js'
import type {
  ListMeteringEventsRequest,
  ListMeteringEventsResponse,
  IngestMeteringEventsRequest,
  IngestMeteringEventsResponse,
} from '../models/operations/events.js'

export class Events {
  constructor(private readonly _client: Client) {}

  async list(
    request?: ListMeteringEventsRequest,
    options?: RequestOptions,
  ): Promise<ListMeteringEventsResponse> {
    return unwrap(await listMeteringEvents(this._client, request, options))
  }

  async ingest(
    request: IngestMeteringEventsRequest,
    options?: RequestOptions,
  ): Promise<IngestMeteringEventsResponse> {
    return unwrap(await ingestMeteringEvents(this._client, request, options))
  }
}
