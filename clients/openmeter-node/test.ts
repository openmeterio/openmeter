import assert from 'node:assert'
import { mock, test } from 'node:test'

import { RequestInfo, RequestInit, Response } from 'node-fetch'
import { OpenMeter } from './src/index.js'

test('should ingest event', async () => {
	const mockFetch = mock.fn(
		(url: URL | RequestInfo, init?: RequestInit | undefined) =>
			Promise.resolve(new Response())
	)
	const openmeter = new OpenMeter({
		baseUrl: 'http://localhost:8888',
		fetch: mockFetch,
	})

	const event = {
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
	await openmeter.ingestEvents(event)

	assert.strictEqual(mockFetch.mock.calls.length, 1)
	const call = mockFetch.mock.calls[0]
	assert.deepStrictEqual(
		call.arguments[0],
		'http://localhost:8888/api/v1alpha1/events'
	)
	assert.deepStrictEqual(call.arguments[1]?.body, JSON.stringify(event))
})

test('should get meter', async () => {
	const mockFetch = mock.fn(
		(url: URL | RequestInfo, init?: RequestInit | undefined) =>
			Promise.resolve(new Response('mock-data'))
	)
	const openmeter = new OpenMeter({
		baseUrl: 'http://localhost:8888',
		fetch: mockFetch,
	})

	const { data } = await openmeter.getMetersById({ meterId: 'm1' })

	assert.strictEqual(mockFetch.mock.calls.length, 1)
	const call = mockFetch.mock.calls[0]
	assert.deepStrictEqual(
		call.arguments[0],
		'http://localhost:8888/api/v1alpha1/meters/m1'
	)
	assert.deepStrictEqual(data, 'mock-data')
})
