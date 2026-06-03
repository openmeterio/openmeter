import type { Client } from 'openapi-fetch'
import type { RequestOptions } from '../client/common.js'
import { transformResponse } from '../client/utils.js'
import type { MeteringEvent, operations, paths } from './schemas.js'

/**
 * Metering Events (v3)
 *
 * Thin wrapper over the v3 events endpoints. Bodies use the v3 wire shape
 * verbatim (snake_case); no field renaming (Option A).
 */
export class Events {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Ingest a metering event or batch of events (CloudEvents). Returns 202 with
   * no body on success.
   */
  public async ingest(
    events: MeteringEvent | MeteringEvent[],
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST('/openmeter/events', {
      body: events,
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List ingested metering events
   */
  public async list(
    params?: operations['list-metering-events']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/openmeter/events', {
      params: { query: params },
      ...options,
    })

    return transformResponse(resp)
  }
}
