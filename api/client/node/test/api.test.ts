import crypto from 'crypto'
import undici from 'undici'
import { vi, describe, it, expect, beforeEach } from 'vitest'
// test built version
import { OpenMeter, type Event } from '../dist/index.js'

declare module 'vitest' {
	export interface TestContext {
		openmeter: OpenMeter
	}
}

vi.spyOn(undici, 'request').getMockImplementation()
vi.spyOn(crypto, 'randomUUID').mockReturnValue('aaf17be7-860c-4519-91d3-00d97da3cc65')

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
				time: new Date(),
				data: {
					api_calls: 1,
				},
			}
			await openmeter.ingestEvents(event)

			expect(undici.request).toHaveBeenCalledWith(
				new URL('/api/v1/events', 'http://127.0.0.1:8888'),
				{
					method: 'POST',
					body: JSON.stringify(event),
					headers: {
						Accept: 'application/json',
						'Content-Type': 'application/cloudevents+json',
					},
				}
			)
		})

		it('should ingest event with defaults', async ({ openmeter }) => {
			const event: Event = {
				type: 'my-type',
				subject: 'my-awesome-user-id',
				data: {
					api_calls: 1,
				},
			}
			await openmeter.ingestEvents(event)

			expect(undici.request).toHaveBeenCalledWith(
				new URL('/api/v1/events', 'http://127.0.0.1:8888'),
				{
					method: 'POST',
					body: JSON.stringify({
						specversion: '1.0',
						id: 'aaf17be7-860c-4519-91d3-00d97da3cc65',
						source: '@openmeter/sdk',
						type: 'my-type',
						subject: 'my-awesome-user-id',
						data: {
							api_calls: 1,
						},
					}),
					headers: {
						Accept: 'application/json',
						'Content-Type': 'application/cloudevents+json',
					},
				}
			)
		})
	})

	// describe('listMeters', () => {
	// 	it('should list meters', async ({ openmeter }) => {
	// 		await openmeter.meters.listMeters()

	// 		expect(fetch).toHaveBeenCalledOnce()
	// 		expect(fetch).toHaveBeenCalledWith(
	// 			'http://127.0.0.1:8888/api/v1/meters',
	// 			expect.objectContaining({
	// 				method: 'GET',
	// 				headers: new Headers({
	// 					Accept: 'application/json',
	// 				}),
	// 			})
	// 		)
	// 	})
	// })

	// describe('getMeter', () => {
	// 	it('should get meter', async ({ openmeter }) => {
	// 		const meterSlug = 'm1'
	// 		await openmeter.meters.getMeter(meterSlug)

	// 		expect(fetch).toHaveBeenCalledOnce()
	// 		expect(fetch).toHaveBeenCalledWith(
	// 			'http://127.0.0.1:8888/api/v1/meters/m1',
	// 			expect.objectContaining({
	// 				method: 'GET',
	// 				headers: new Headers({
	// 					Accept: 'application/json',
	// 				}),
	// 			})
	// 		)
	// 	})
	// })

	// describe('getMeterValues', () => {
	// 	it('should get meter values', async ({ openmeter }) => {
	// 		const meterSlug = 'm1'
	// 		await openmeter.meters.getMeterValues(meterSlug)

	// 		expect(fetch).toHaveBeenCalledOnce()
	// 		expect(fetch).toHaveBeenCalledWith(
	// 			'http://127.0.0.1:8888/api/v1/meters/m1/values',
	// 			expect.objectContaining({
	// 				method: 'GET',
	// 				headers: new Headers({
	// 					Accept: 'application/json',
	// 				}),
	// 			})
	// 		)
	// 	})

	// 	it('should get meter values (with params)', async ({ openmeter }) => {
	// 		const meterSlug = 'm2'
	// 		const subject = 'user-1'
	// 		const from = new Date('2021-01-01')
	// 		const to = new Date('2021-01-02')
	// 		const windowSize = WindowSize.HOUR
	// 		await openmeter.meters.getMeterValues(
	// 			meterSlug,
	// 			undefined,
	// 			subject,
	// 			from,
	// 			to,
	// 			windowSize
	// 		)

	// 		expect(fetch).toHaveBeenCalledOnce()
	// 		expect(fetch).toHaveBeenCalledWith(
	// 			'http://127.0.0.1:8888/api/v1/meters/m2/values?subject=user-1&from=2021-01-01T00%3A00%3A00.000Z&to=2021-01-02T00%3A00%3A00.000Z&windowSize=HOUR',
	// 			expect.objectContaining({
	// 				method: 'GET',
	// 				headers: new Headers({
	// 					Accept: 'application/json',
	// 				}),
	// 			})
	// 		)
	// 	})
	// })
})
