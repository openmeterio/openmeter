import { request } from 'undici'
import { paths, components } from '../schemas/openapi.js'
import { BaseClient, HttpError, OpenMeterConfig, Problem, RequestOptions } from './client.js'

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

export type MeterAggregation = components['schemas']['MeterAggregation']
export type WindowSize = components['schemas']['WindowSize']
export type MeterValue = components['schemas']['MeterValue']
export type Meter = components['schemas']['Meter']


export class MetersClient extends BaseClient {
    constructor(config: OpenMeterConfig) {
        super(config)
    }

    public async retreive(slug: string, options?: RequestOptions): Promise<Meter> {
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

    public async list(options?: RequestOptions): Promise<Meter[]> {
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

    public async query(slug: string, params?: MeterQueryParams, options?: RequestOptions): Promise<MeterQueryResponse> {
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
}

