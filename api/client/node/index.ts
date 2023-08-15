import crypto from 'crypto'
import { IncomingHttpHeaders } from 'http'
import { request } from 'undici'
import { components, paths } from './schemas/openapi.js'

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

export type MeterQueryParams = {
    subject?: string
    /**
     * @description Start date.
     * Must be aligned with the window size.
     * Inclusive.
     */
    from?: Date
    /**
     * @description End date.
     * Must be aligned with the window size.
     * Inclusive.
     */
    to?: Date
    /** @description If not specified, a single usage aggregate will be returned for the entirety of the specified period for each subject and group. */
    windowSize?: WindowSize
    /** @description If not specified a single aggregate will be returned for each subject and time window. */
    groupBy?: string[]
}

export type MeterQueryResponse = paths['/api/v1/meters/{meterIdOrSlug}/values']['get']['responses']['200']['content']['application/json']

export type Problem = components['schemas']['Problem']
export type MeterAggregation = components['schemas']['MeterAggregation']
export type WindowSize = components['schemas']['WindowSize']
export type MeterValue = components['schemas']['MeterValue']
export type Meter = components['schemas']['Meter']

// We export Event instead
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
        if (resp.statusCode > 299) {
            const problem = (await resp.body.json()) as Problem

            throw new HttpError('unexpected status code', {
                statusCode: resp.statusCode,
                problem,
            })
        }
    }

    public async retrieveMeter(slug: string, options?: RequestOptions): Promise<Meter> {
        const url = new URL(`/api/v1/meters/${slug}`, this.config.baseUrl)
        const resp = await request(url, {
            method: 'GET',
            headers: {
                Accept: 'application/json',
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
        const body = await resp.body.json() as Meter
        return body
    }

    public async listMeters(options?: RequestOptions): Promise<Meter[]> {
        const url = new URL('/api/v1/meters', this.config.baseUrl)
        const resp = await request(url, {
            method: 'GET',
            headers: {
                Accept: 'application/json',
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
        const body = await resp.body.json() as Meter[]
        return body
    }

    public async queryMeter(slug: string, params?: MeterQueryParams, options?: RequestOptions): Promise<MeterQueryResponse> {
        // Making Request
        const searchParams = new URLSearchParams()
        if (params && params.from) {
            searchParams.append('from', params.from.toISOString())
        }
        if (params && params.to) {
            searchParams.append('to', params.to.toISOString())
        }
        if (params && params.subject) {
            searchParams.append('subject', params.subject)
        }
        if (params && params.groupBy) {
            searchParams.append('groupBy', params.groupBy.join(','))
        }
        if (params && params.windowSize) {
            searchParams.append('windowSize', params.windowSize)
        }
        let qs = searchParams.toString()
        qs = qs.length > 0 ? `?${qs}` : ''
        const url = new URL(`/api/v1/meters/${slug}/values${qs}`, this.config.baseUrl)
        const resp = await request(url, {
            method: 'GET',
            headers: {
                Accept: 'application/json',
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
        const body = await resp.body.json() as MeterQueryResponse
        return body
    }

    private authHeaders(): IncomingHttpHeaders {
        if (this.config.token) {
            return {
                authorization: `Bearer ${this.config.token} `,
            }
        }

        if (this.config.username && this.config.password) {
            const encoded = Buffer.from(
                `${this.config.username}:${this.config.password} `
            ).toString('base64')
            return {
                authorization: `Basic ${encoded} `,
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
