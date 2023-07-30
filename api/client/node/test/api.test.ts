import * as nodeFetch from 'node-fetch'
import fetch, { Response, Headers } from 'node-fetch'
import { vi, describe, it, expect, beforeEach } from 'vitest'
// test built version
import { OpenMeter, WindowSize, type Event } from '../dist/index.js'

declare module 'vitest' {
	export interface TestContext {
		openmeter: OpenMeter
	}
}

vi.mock('node-fetch', async () => {
	const actual: typeof nodeFetch = await vi.importActual('node-fetch')

	return {
		...actual,
		default: vi.fn().mockImplementation(() => Promise.resolve(new Response())),
	}
})

describe('api', () => {
	beforeEach((ctx) => {
		vi.clearAllMocks()

		ctx.openmeter = new OpenMeter({
			baseUrl: 'http://127.0.0.1:8888',
		})
	})

	describe('ingestEvents', () => {
		it('should ingest event', async ({ openmeter }) => {
			const event: Event = {
				specversion: '1.0',
				id: 'id-1',
				source: 'my-app',
				type: 'my-type',
				subject: 'my-awesome-user-id',
				time: new Date().toISOString(),
				data: {
					api_calls: 1,
				},
			}
			await openmeter.events.ingestEvents(event)

			expect(fetch).toHaveBeenCalledOnce()
			expect(fetch).toHaveBeenCalledWith(
				'http://127.0.0.1:8888/api/v1/events',
				expect.objectContaining({
					method: 'POST',
					body: JSON.stringify(event),
					headers: new Headers({
						Accept: 'application/json',
						'Content-Type': 'application/cloudevents+json',
					}),
				})
			)
		})
	})

	describe('listMeters', () => {
		it('should list meters', async ({ openmeter }) => {
			await openmeter.meters.listMeters()

			expect(fetch).toHaveBeenCalledOnce()
			expect(fetch).toHaveBeenCalledWith(
				'http://127.0.0.1:8888/api/v1/meters',
				expect.objectContaining({
					method: 'GET',
					headers: new Headers({
						Accept: 'application/json',
					}),
				})
			)
		})
	})

	describe('getMeter', () => {
		it('should get meter', async ({ openmeter }) => {
			const meterSlug = 'm1'
			await openmeter.meters.getMeter(meterSlug)

			expect(fetch).toHaveBeenCalledOnce()
			expect(fetch).toHaveBeenCalledWith(
				'http://127.0.0.1:8888/api/v1/meters/m1',
				expect.objectContaining({
					method: 'GET',
					headers: new Headers({
						Accept: 'application/json',
					}),
				})
			)
		})
	})

	describe('getMeterValues', () => {
		it('should get meter values', async ({ openmeter }) => {
			const meterSlug = 'm1'
			await openmeter.meters.getMeterValues(meterSlug)

			expect(fetch).toHaveBeenCalledOnce()
			expect(fetch).toHaveBeenCalledWith(
				'http://127.0.0.1:8888/api/v1/meters/m1/values',
				expect.objectContaining({
					method: 'GET',
					headers: new Headers({
						Accept: 'application/json',
					}),
				})
			)
		})

		it('should get meter values (with params)', async ({ openmeter }) => {
			const meterSlug = 'm2'
			const subject = 'user-1'
			const from = new Date('2021-01-01').toISOString()
			const to = new Date('2021-01-02').toISOString()
			const windowSize = WindowSize.HOUR
			await openmeter.meters.getMeterValues(
				meterSlug,
				undefined,
				subject,
				from,
				to,
				windowSize
			)

			expect(fetch).toHaveBeenCalledOnce()
			expect(fetch).toHaveBeenCalledWith(
				'http://127.0.0.1:8888/api/v1/meters/m2/values?subject=user-1&from=2021-01-01T00%3A00%3A00.000Z&to=2021-01-02T00%3A00%3A00.000Z&windowSize=HOUR',
				expect.objectContaining({
					method: 'GET',
					headers: new Headers({
						Accept: 'application/json',
					}),
				})
			)
		})
	})
})
