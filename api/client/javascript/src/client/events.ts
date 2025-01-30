import crypto from 'crypto'
import { transformResponse } from './utils.js'
import type { RequestOptions } from './common.js'
import type { operations, paths, Event } from './schemas.js'
import type { Client } from 'openapi-fetch'

/**
 * Events are used to track usage of your product or service.
 * Events are processed asynchronously by the meters, so they may not be immediately available for querying.
 */
export class Events {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Ingests an event or batch of events
   * @param events - The events to ingest
   * @param signal - An optional abort signal
   * @returns The ingested events
   */
  public async ingest(events: Event | Event[], options?: RequestOptions) {
    const body = (Array.isArray(events) ? events : [events]).map(
      setDefaultsForEvent
    )

    const resp = await this.client.POST('/api/v1/events', {
      body,
      headers: {
        'Content-Type': 'application/cloudevents-batch+json',
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List ingested events
   * @param query - The query parameters
   * @param signal - An optional abort signal
   * @returns The events
   */
  public async list(
    query?: operations['listEvents']['parameters']['query'],
    options?: RequestOptions
  ) {
    const resp = await this.client.GET('/api/v1/events', {
      params: {
        query,
      },
      ...options,
    })

    return transformResponse(resp)
  }
}

export function setDefaultsForEvent(ev: Event): Event {
  return {
    data: ev.data,
    datacontenttype: ev.datacontenttype,
    dataschema: ev.dataschema,
    id: ev.id ?? crypto.randomUUID(),
    source: ev.source ?? '@openmeter/sdk',
    specversion: ev.specversion ?? '1.0',
    subject: ev.subject,
    time: ev.time,
    type: ev.type,
  }
}
