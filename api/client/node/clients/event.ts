import crypto from 'crypto'
import { RequestOptions } from 'http'
import { request } from 'undici'
import { components } from '../schemas/openapi.js'
import { BaseClient, HttpError, OpenMeterConfig, Problem } from './client.js'

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

    public async ingest(
        usageEvent: Event,
        options?: RequestOptions
    ): Promise<void> {
        // We default where we can to lower the barrier to use CloudEvents
        const payload: CloudEvents = {
            specversion: usageEvent.specversion ?? '1.0',
            id: usageEvent.id ?? crypto.randomUUID(),
            source: usageEvent.source ?? '@openmeter/sdk',
            type: usageEvent.type,
            subject: usageEvent.subject,
        }

        // Optional fields
        if (usageEvent.time) {
            payload.time = usageEvent.time.toISOString()
        }
        if (usageEvent.data) {
            payload.data = usageEvent.data
        }
        if (usageEvent.dataschema) {
            payload.dataschema = usageEvent.dataschema
        }
        if (usageEvent.datacontenttype) {
            if (usageEvent.datacontenttype !== 'application/json') {
                throw new TypeError(
                    `Unsupported datacontenttype: ${usageEvent.datacontenttype}`
                )
            }

            payload.datacontenttype = usageEvent.datacontenttype
        }

        // Making Request
        const url = new URL('/api/v1/events', this.config.baseUrl)
        const resp = await request(url, {
            method: 'POST',
            body: JSON.stringify(payload),
            headers: {
                Accept: 'application/json',
                'Content-Type': 'application/cloudevents+json',
                ...this.authHeaders(),
                ...this.config.headers,
                ...options?.headers,
            },
        })
        if (resp.statusCode > 399) {
            const problem = (await resp.body.json()) as Problem

            throw new HttpError({
                statusCode: resp.statusCode,
                problem,
            })
        }
    }
}

