import { IncomingHttpHeaders } from 'http'
import { Dispatcher, request } from 'undici'
import { components } from '../schemas/openapi.js'

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

export type Problem = components['schemas']['Problem']

type UndiciRequestOptions = { dispatcher?: Dispatcher } & Omit<Dispatcher.RequestOptions, 'origin' | 'path' | 'method'> & Partial<Pick<Dispatcher.RequestOptions, 'method'>>

export class BaseClient {
    protected config: OpenMeterConfig

    constructor(config: OpenMeterConfig) {
        this.config = config
    }

    protected async request<T>({
        path,
        method,
        searchParams,
        headers,
        body,
        options
    }: {
        path: string
        method: Dispatcher.HttpMethod,
        searchParams?: URLSearchParams,
        headers?: IncomingHttpHeaders,
        body?: string | Buffer | Uint8Array,
        options?: RequestOptions
    }): Promise<T> {
        // Building URL
        const url = this.getUrl(path, searchParams)

        // Request options
        const reqHeaders: IncomingHttpHeaders = {
            Accept: 'application/json',
            ...headers,
            ...this.getAuthHeaders(),
            ...this.config.headers,
            ...options?.headers,
        }
        const reqOpts: UndiciRequestOptions = {
            method,
            headers: reqHeaders
        }

        // Optional body
        if (body) {
            if (!reqHeaders['Content-Type'] && !reqHeaders['content-type']) {
                throw new Error('Content Type is required with body')
            }

            reqOpts.body = body
        }

        const resp = await request(url, reqOpts)

        // Error handling
        if (resp.statusCode > 399) {
            if (resp.headers['content-type'] === 'application/problem+json') {
                const problem = await resp.body.json() as Problem
                throw new HttpError({
                    statusCode: resp.statusCode,
                    problem,
                })
            }

            // Requests can fail before API, in this case we only have a status code
            throw new HttpError({
                statusCode: resp.statusCode,
            })
        }

        // Response parsing
        if (resp.statusCode === 204) {
            return undefined as unknown as T
        }
        if (resp.headers['content-type'] === 'application/json') {
            return await resp.body.json() as T
        }
        if (!resp.headers['content-type']) {
            throw new Error('Missing content type')
        }

        throw new Error(`Unknown content type: ${resp.headers['content-type']}`)
    }

    protected getUrl(path: string, searchParams?: URLSearchParams) {
        let qs = searchParams ? searchParams.toString() : ''
        qs = qs.length > 0 ? `?${qs}` : ''
        const url = new URL(`${path}${qs}`, this.config.baseUrl)
        return url
    }

    protected getAuthHeaders(): IncomingHttpHeaders {
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

export class HttpError extends Error {
    public statusCode: number
    public problem?: Problem

    constructor({ statusCode, problem }: { statusCode: number; problem?: Problem }) {
        super(problem?.type || 'unexpected status code')
        this.name = 'HttpError'
        this.statusCode = statusCode
        this.problem = problem
    }
}
