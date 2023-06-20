import createClient from 'openapi-fetch'
import type fetch from 'node-fetch'
import type { RequestInit } from 'node-fetch'
import type { paths, components, operations } from '../openapi.d.ts'

export class OpenMeter {
	private client: ReturnType<typeof createClient<paths>>

	constructor(opts: RequestInit & { baseUrl: string; fetch?: typeof fetch }) {
		this.client = createClient<paths>(opts)
	}

	public async ingestEvents(
		event: components['schemas']['Event'],
		opts?: RequestInit
	) {
		const { data, response, error } = await this.client.post(
			'/api/v1alpha1/events',
			{
				...opts,
				headers: {
					'content-type': 'application/cloudevents+json',
					...opts?.headers,
				},
				body: event,
			}
		)
		if (error) {
			throw new HttpError(error)
		}

		return { data, response }
	}

	public async getMeters(opts?: RequestInit) {
		const { data, response, error } = await this.client.get(
			'/api/v1alpha1/meters',
			{
				...opts,
				headers: {
					'content-type': 'application/json',
					...opts?.headers,
				},
			}
		)
		if (error) {
			throw new HttpError(error)
		}

		return { data, response }
	}

	public async getMetersById(
		path: operations['getMetersById']['parameters']['path'],
		opts?: RequestInit
	) {
		const { data, response, error } = await this.client.get(
			'/api/v1alpha1/meters/{meterId}',
			{
				...opts,
				params: { path },
				headers: {
					'content-type': 'application/json',
					...opts?.headers,
				},
			}
		)
		if (error) {
			throw new HttpError(error)
		}

		return { data, response }
	}

	public async getValuesByMeterId(
		path: operations['getValuesByMeterId']['parameters']['path'],
		query: operations['getValuesByMeterId']['parameters']['query'],
		opts?: RequestInit
	) {
		const { data, response, error } = await this.client.get(
			'/api/v1alpha1/meters/{meterId}/values',
			{
				...opts,
				params: { path, query },
				headers: {
					'content-type': 'application/json',
					...opts?.headers,
				},
			}
		)
		if (error) {
			throw new HttpError(error)
		}

		return { data, response }
	}
}

export class HttpError extends Error {
	public statusCode?: number
	public status?: string
	public code?: number

	constructor(
		params: {
			statusCode?: number | undefined
			status?: string | undefined
			code?: number | undefined
			message?: string | undefined
		},
		options?: ErrorOptions
	) {
		super(params.message ?? 'HttpError', options)

		this.name = 'HttpError'
		this.statusCode = params.statusCode
		this.status = params.status
		this.code = params.code
	}
}
