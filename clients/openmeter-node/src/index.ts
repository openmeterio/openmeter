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
		return await this.client.post('/api/v1alpha1/events', {
			...opts,
			headers: {
				'content-type': 'application/cloudevents+json',
				...opts?.headers,
			},
			body: event,
		})
	}

	public async getMeters(opts?: RequestInit) {
		return await this.client.get('/api/v1alpha1/meters', {
			...opts,
			headers: {
				'content-type': 'application/json',
				...opts?.headers,
			},
		})
	}

	public async getMetersById(
		path: operations['getMetersById']['parameters']['path'],
		opts?: RequestInit
	) {
		return await this.client.get('/api/v1alpha1/meters/{meterId}', {
			...opts,
			params: { path },
			headers: {
				'content-type': 'application/json',
				...opts?.headers,
			},
		})
	}

	public async getValuesByMeterId(
		path: operations['getValuesByMeterId']['parameters']['path'],
		query: operations['getValuesByMeterId']['parameters']['query'],
		opts?: RequestInit
	) {
		return await this.client.get('/api/v1alpha1/meters/{meterId}/values', {
			...opts,
			params: { path, query },
			headers: {
				'content-type': 'application/json',
				...opts?.headers,
			},
		})
	}
}
