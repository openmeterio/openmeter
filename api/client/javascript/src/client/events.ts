import crypto from 'crypto'
import { transformResponse, type RequestOptions } from './utils.js'
import type { operations, paths, Event as SchemaEvent } from './schemas.js'
import type { Client } from 'openapi-fetch'

/**
 * Usage Event
 */
export type Event = {
  /**
   * @description The version of the CloudEvents specification which the event uses.
   * @example 1.0
   */
  specversion?: string
  /**
   * @description Unique identifier for the event, defaults to uuid v4.
   * @example "5c10fade-1c9e-4d6c-8275-c52c36731d3c"
   */
  id?: string
  /**
   * Format: uri-reference
   * @description Identifies the context in which an event happened, defaults to: @openmeter/sdk
   * @example services/service-0
   */
  source?: string
  /**
   * @description Describes the type of event related to the originating occurrence.
   * @example "api_request"
   */
  type: string
  /**
   * @description Describes the subject of the event in the context of the event producer (identified by source).
   * @example "customer_id"
   */
  subject: string
  /**
   * Format: date-time
   * @description Date of when the occurrence happened.
   * @example new Date('2023-01-01T01:01:01.001Z')
   */
  time?: Date
  /**
   * Format: uri
   * @description Identifies the schema that data adheres to.
   */
  dataschema?: string
  /**
   * @description Content type of the data value. Must adhere to RFC 2046 format.
   * @example application/json
   * @enum {string|null}
   */
  datacontenttype?: 'application/json'
  /**
   * @description The event payload.
   * @example {
   *   "duration_ms": "12",
   *   "path": "/hello"
   * }
   */
  data: Record<string, unknown>
}

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
    const body = (Array.isArray(events) ? events : [events]).map(transformEvent)

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

function transformEvent(ev: Event): SchemaEvent {
  return {
    data: ev.data,
    datacontenttype: ev.datacontenttype ?? 'application/json',
    dataschema: ev.dataschema,
    id: ev.id ?? crypto.randomUUID(),
    source: ev.source ?? '@openmeter/sdk',
    specversion: ev.specversion ?? '1.0',
    subject: ev.subject,
    time: ev.time,
    type: ev.type,
  }
}
