import crypto from 'crypto'
import { components } from '../schemas/openapi.js'
import { RequestOptions, BaseClient, OpenMeterConfig } from './client.js'

// We export Event instead
export type CloudEvent = components['schemas']['Event']
export type IngestedEvent = components['schemas']['IngestedEvent']

export type EventsQueryParams = {
  /**
   * @description Limit number of results. Max: 100
   * @example 25
   */
  limit?: number
}

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
  data: Record<string, string | number>
}

export class EventsClient extends BaseClient {
  constructor(config: OpenMeterConfig) {
    super(config)
  }

  /**
   * Ingest usage event in a CloudEvents format
   * @see https://cloudevents.io
   */
  public async ingest(
    usageEvent: Event | Event[],
    options?: RequestOptions
  ): Promise<void> {
    const isBatch = Array.isArray(usageEvent)
    const cloudEvents: CloudEvent[] = (isBatch ? usageEvent : [usageEvent]).map(
      (ev) => {
        // Validate content type
        if (ev.datacontenttype && ev.datacontenttype !== 'application/json') {
          throw new TypeError(
            `Unsupported datacontenttype: ${ev.datacontenttype}`
          )
        }

        // We default where we can to lower the barrier to use CloudEvents
        const cloudEvent: CloudEvent = {
          specversion: ev.specversion ?? '1.0',
          id: ev.id ?? crypto.randomUUID(),
          source: ev.source ?? '@openmeter/sdk',
          type: ev.type,
          subject: ev.subject,
          time: ev.time?.toISOString(),
          datacontenttype: ev.datacontenttype,
          dataschema: ev.dataschema,
          data: ev.data,
        }

        return cloudEvent
      }
    )

    const contentType = isBatch
      ? 'application/cloudevents-batch+json'
      : 'application/cloudevents+json'
    const body = isBatch
      ? JSON.stringify(cloudEvents)
      : JSON.stringify(cloudEvents[0])

    // Making Request
    return await this.request({
      path: '/api/v1/events',
      method: 'POST',
      body,
      headers: {
        'Content-Type': contentType,
      },
      options,
    })
  }

  /**
   * List raw events
   */
  public async list(
    params?: EventsQueryParams,
    options?: RequestOptions
  ): Promise<IngestedEvent[]> {
    const searchParams = params
      ? BaseClient.toURLSearchParams(params)
      : undefined
    return this.request<IngestedEvent[]>({
      method: 'GET',
      path: `/api/v1/events`,
      searchParams,
      options,
    })
  }
}
