import { IncomingHttpHeaders } from 'http'
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

export class BaseClient {
    protected config: OpenMeterConfig

    constructor(config: OpenMeterConfig) {
        this.config = config
    }

    protected authHeaders(): IncomingHttpHeaders {
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
