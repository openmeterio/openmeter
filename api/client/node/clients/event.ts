import crypto from 'crypto'
import { components } from '../schemas/openapi.js'
import { RequestOptions, BaseClient, OpenMeterConfig } from './client.js'

// We export Event instead
type CloudEvents = components['schemas']['Event']

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
    data: Record<string, string | number | Record<string, string | number>>
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
        usageEvent: Event,
        options?: RequestOptions
    ): Promise<void> {
        // We default where we can to lower the barrier to use CloudEvents
        const body: CloudEvents = {
            specversion: usageEvent.specversion ?? '1.0',
            id: usageEvent.id ?? crypto.randomUUID(),
            source: usageEvent.source ?? '@openmeter/sdk',
            type: usageEvent.type,
            subject: usageEvent.subject,
        }

        // Optional fields
        if (usageEvent.time) {
            body.time = usageEvent.time.toISOString()
        }
        if (usageEvent.data) {
            body.data = usageEvent.data
        }
        if (usageEvent.dataschema) {
            body.dataschema = usageEvent.dataschema
        }
        if (usageEvent.datacontenttype) {
            if (usageEvent.datacontenttype !== 'application/json') {
                throw new TypeError(
                    `Unsupported datacontenttype: ${usageEvent.datacontenttype}`
                )
            }

            body.datacontenttype = usageEvent.datacontenttype
        }

        // Making Request
        return await this.request({
            path: '/api/v1/events',
            method: 'POST',
            body: JSON.stringify(body),
            headers: {
                'Content-Type': 'application/cloudevents+json',
            },
            options
        })
    }
}

