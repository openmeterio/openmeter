import crypto from 'crypto'
import { IncomingHttpHeaders } from 'http'
import undici from 'undici'
import { components } from './schemas/openapi.js'

export type OpenMeterConfig = {
    baseUrl: string
    token?: string
    username?: string
    password?: string
    headers?: IncomingHttpHeaders
}

export type RequestOptions = {
    headers?: IncomingHttpHeaders
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
    data: Record<string, string | number | Record<string, string | number>>
}

export type Problem = components['schemas']['Problem']
export type MeterAggregation = components['schemas']['MeterAggregation']
export type WindowSize = components['schemas']['WindowSize']

type CloudEvents = components['schemas']['Event']

export class OpenMeter {
    private config: OpenMeterConfig

    constructor(config: OpenMeterConfig) {
        this.config = config
    }

    public async ingestEvents(
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
        const resp = await undici.request(url, {
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
        if (resp.statusCode > 299) {
            const problem = (await resp.body.json()) as Problem

            throw new HttpError('unexpected status code', {
                statusCode: resp.statusCode,
                problem,
            })
        }
    }

    private authHeaders(): IncomingHttpHeaders {
        if (this.config.token) {
            return {
                authorization: `Bearer ${this.config.token}`,
            }
        }

        if (this.config.username && this.config.password) {
            const encoded = Buffer.from(
                `${this.config.username}:${this.config.password}`
            ).toString('base64')
            return {
                authorization: `Basic ${encoded}`,
            }
        }

        return {}
    }
}

class HttpError extends Error {
    public statusCode: number
    public problem: Problem

    constructor(
        message: string,
        { statusCode, problem }: { statusCode: number; problem: Problem }
    ) {
        super(message)
        this.name = 'HttpError'
        this.statusCode = statusCode
        this.problem = problem
    }
}
