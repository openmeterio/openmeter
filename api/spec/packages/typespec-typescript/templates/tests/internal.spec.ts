import fetchMock from '@fetch-mock/vitest'
import { beforeEach, describe, expect, it } from 'vitest'
import { OpenMeter } from '../src/index.js'

beforeEach(() => {
  fetchMock.mockReset()
})

const fetch = fetchMock.fetchHandler

function client(): OpenMeter {
  return new OpenMeter({
    baseUrl: 'https://eu.api.konghq.com/v3',
    apiKey: 'k',
    fetch,
  })
}

function lastUrl(): string {
  return fetchMock.callHistory.lastCall()!.url
}

describe('internal sub-client', () => {
  it('is memoized and separate from the public surface', () => {
    const sdk = client()
    expect(sdk.internal).toBe(sdk.internal)
    // x-internal operations must not leak onto the public sub-clients.
    expect('void' in sdk.customers.credits.grants).toBe(false)
    // Entirely-internal groups have no public getter at all.
    expect('currencies' in sdk).toBe(false)
  })

  it('routes internal.customers.credits.grants.void() to the credit grant void action', async () => {
    fetchMock.route('*', {
      body: {
        id: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
        name: 'Grant',
        funding_method: 'none',
        currency: 'USD',
        amount: '100',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      },
      headers: { 'Content-Type': 'application/json' },
    })
    await client().internal.customers.credits.grants.void({
      customerId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
      creditGrantId: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
      body: {},
    })
    expect(lastUrl()).toBe(
      'https://eu.api.konghq.com/v3/openmeter/customers/01ARZ3NDEKTSV4RRFFQ69G5FAV/credits/grants/01ARZ3NDEKTSV4RRFFQ69G5FAW/void',
    )
  })

  it('routes internal.currencies.list() to the currencies resource', async () => {
    fetchMock.route('*', {
      body: { data: [], meta: { page: { number: 1, size: 10, total: 0 } } },
      headers: { 'Content-Type': 'application/json' },
    })
    await client().internal.currencies.list()
    expect(lastUrl()).toBe('https://eu.api.konghq.com/v3/openmeter/currencies')
  })
})
