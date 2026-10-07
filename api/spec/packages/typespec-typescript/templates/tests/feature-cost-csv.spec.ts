import fetchMock from '@fetch-mock/vitest'
import { beforeEach, expect, it } from 'vitest'
import { Client, funcs } from '../src/index.js'

beforeEach(() => fetchMock.mockReset())

it('feature cost CSV sends the shared JSON query body and returns CSV text', async () => {
  const csv =
    'from,to,usage,cost,currency,detail\n2024-01-01T00:00:00Z,2024-01-02T00:00:00Z,100,0.001,USD,\n'
  fetchMock.route('*', { body: csv, headers: { 'Content-Type': 'text/csv' } })
  const client = new Client({
    baseUrl: 'https://api.example.com',
    apiKey: 'test-key',
    fetch: fetchMock.fetchHandler,
    validate: true,
  })

  const result = await funcs.queryFeatureCostCsv(client, {
    featureId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
    body: {
      from: '2024-01-01T00:00:00Z',
      to: '2024-01-02T00:00:00Z',
      timeZone: 'UTC',
      groupByDimensions: ['model'],
    },
  })

  expect(result.ok).toBe(true)
  expect(result.value).toBe(csv)
  const call = fetchMock.callHistory.lastCall()!
  expect(call.url).toBe(
    'https://api.example.com/openmeter/features/01ARZ3NDEKTSV4RRFFQ69G5FAV/cost/query',
  )
  expect(new Headers(call.options.headers).get('Accept')).toBe('text/csv')
  expect(new Headers(call.options.headers).get('Content-Type')).toContain(
    'application/json',
  )
  expect(JSON.parse(call.options.body as string)).toEqual({
    from: '2024-01-01T00:00:00Z',
    to: '2024-01-02T00:00:00Z',
    time_zone: 'UTC',
    group_by_dimensions: ['model'],
  })
})
