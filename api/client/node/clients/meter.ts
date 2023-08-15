import { paths, components } from '../schemas/openapi.js'
import { BaseClient, OpenMeterConfig, RequestOptions } from './client.js'

export enum WindowSize {
    MINUTE = 'MINUTE',
    HOUR = 'HOUR',
    DAY = 'DAY'
}

export enum MeterAggregation {
    SUM = 'SUM',
    COUNT = 'COUNT',
    AVG = 'AVG',
    MIN = 'MIN',
    MAX = 'MAX',
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
    windowSize?: WindowSizeType
    /** @description If not specified a single aggregate will be returned for each subject and time window. */
    groupBy?: string[]
}

export type MeterQueryResponse = paths['/api/v1/meters/{meterIdOrSlug}/values']['get']['responses']['200']['content']['application/json']

export type MeterValue = components['schemas']['MeterValue']
export type Meter = components['schemas']['Meter']
export type WindowSizeType = components['schemas']['WindowSize']

export class MetersClient extends BaseClient {
    constructor(config: OpenMeterConfig) {
        super(config)
    }

    /**
     * Get one meter by slug
     */
    public async get(slug: string, options?: RequestOptions): Promise<Meter> {
        return this.request<Meter>({
            method: 'GET',
            path: `/api/v1/meters/${slug}`,
            options,
        })
    }

    /**
     * List meters
     */
    public async list(options?: RequestOptions): Promise<Meter[]> {
        return this.request<Meter[]>({
            method: 'GET',
            path: `/api/v1/meters`,
            options,
        })
    }

    /**
     * Get aggregated values of a meter
     */
    public async values(slug: string, params?: MeterQueryParams, options?: RequestOptions): Promise<MeterQueryResponse> {
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
        return this.request<MeterQueryResponse>({
            method: 'GET',
            path: `/api/v1/meters/${slug}/values`,
            searchParams,
            options,
        })
    }
}

