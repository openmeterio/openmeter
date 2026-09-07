import fetchMock from '@fetch-mock/vitest'
import { beforeEach, describe, expect, it } from 'vitest'
import { OpenMeter } from '../src/index.js'

beforeEach(() => {
  fetchMock.mockReset()
})

const fetch = fetchMock.fetchHandler

function lastUrl(): string {
  return fetchMock.callHistory.lastCall()!.url
}

describe('nested sub-clients', () => {
  it('routes internal.customers.charges.list() to the charges sub-resource', async () => {
    fetchMock.route('*', {
      body: { data: [], meta: { page: { number: 1, size: 10, total: 0 } } },
      headers: { 'Content-Type': 'application/json' },
    })
    const sdk = new OpenMeter({
      baseUrl: 'https://eu.api.konghq.com/v3',
      apiKey: 'k',
      fetch,
    })
    await sdk.internal.customers.charges.list({
      customerId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
    })
    expect(lastUrl()).toBe(
      'https://eu.api.konghq.com/v3/openmeter/customers/01ARZ3NDEKTSV4RRFFQ69G5FAV/charges',
    )
  })

  it('serializes charge validation issue presence filters', async () => {
    fetchMock.route('*', {
      body: { data: [], meta: { page: { number: 1, size: 10, total: 0 } } },
      headers: { 'Content-Type': 'application/json' },
    })
    const sdk = new OpenMeter({
      baseUrl: 'https://eu.api.konghq.com/v3',
      apiKey: 'k',
      fetch,
    })

    await sdk.internal.customers.charges.list({
      customerId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
      filter: { validationIssues: '' },
    })
    expect(
      new URL(lastUrl()).searchParams.has('filter[validation_issues]'),
    ).toBe(true)
    expect(
      new URL(lastUrl()).searchParams.get('filter[validation_issues]'),
    ).toBe('')

    await sdk.internal.customers.charges.list({
      customerId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
      filter: { validationIssues: 'null' },
    })
    expect(
      new URL(lastUrl()).searchParams.get('filter[validation_issues]'),
    ).toBe('null')
  })
})
