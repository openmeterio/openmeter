import fetchMock from '@fetch-mock/vitest'
import { beforeEach, describe, expect, it } from 'vitest'
import { OpenMeter, ServerList } from '../src/index.js'
import { SDK_VERSION } from '../src/lib/version.js'

const meter = {
  id: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
  name: 'API calls',
  key: 'api_calls',
  aggregation: 'count',
  event_type: 'api-request',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
}

beforeEach(() => {
  fetchMock.mockReset()
})

function mockMeter() {
  fetchMock.route('*', {
    body: meter,
    headers: { 'Content-Type': 'application/json' },
  })
}

function lastUrl(): string {
  return fetchMock.callHistory.lastCall()!.url
}

function lastAuth(): string | null {
  const headers = fetchMock.callHistory.lastCall()!.options.headers as Headers
  return new Headers(headers).get('authorization')
}

function lastUserAgent(): string | null {
  const headers = fetchMock.callHistory.lastCall()!.options.headers as Headers
  return new Headers(headers).get('user-agent')
}

const fetch = fetchMock.fetchHandler

describe('base URL construction', () => {
  it('preserves the base path segment (/v3)', async () => {
    mockMeter()
    const sdk = new OpenMeter({
      baseUrl: 'https://eu.api.konghq.com/v3',
      apiKey: 'k',
      fetch,
    })
    await sdk.meters.get({ meterId: 'm' })
    expect(lastUrl()).toBe('https://eu.api.konghq.com/v3/openmeter/meters/m')
  })

  it('accepts a URL object as baseUrl', async () => {
    mockMeter()
    const sdk = new OpenMeter({
      baseUrl: new URL('https://us.api.konghq.com/v3'),
      apiKey: 'k',
      fetch,
    })
    await sdk.meters.get({ meterId: 'm' })
    expect(lastUrl()).toBe('https://us.api.konghq.com/v3/openmeter/meters/m')
  })

  it('sets the bearer auth header', async () => {
    mockMeter()
    const sdk = new OpenMeter({
      baseUrl: 'https://eu.api.konghq.com/v3',
      apiKey: 'k',
      fetch,
    })
    await sdk.meters.get({ meterId: 'm' })
    expect(lastAuth()).toBe('Bearer k')
  })
})

describe('server-variable templating', () => {
  it('substitutes a region variable', async () => {
    mockMeter()
    const sdk = new OpenMeter({
      baseUrl: ServerList[0],
      serverVariables: { region: 'eu' },
      apiKey: 'k',
      fetch,
    })
    await sdk.meters.get({ meterId: 'm' })
    expect(lastUrl()).toBe('https://eu.api.konghq.com/v3/openmeter/meters/m')
  })

  it('throws when a required template variable is missing', () => {
    expect(
      () => new OpenMeter({ baseUrl: ServerList[0], apiKey: 'k', fetch }),
    ).toThrow()
  })
})

describe('option clobbering is prevented', () => {
  it('ignores a user-supplied prefix that would redirect requests', async () => {
    mockMeter()
    const sdk = new OpenMeter({
      baseUrl: 'https://eu.api.konghq.com/v3',
      apiKey: 'k',
      prefix: 'http://evil.test/',
      fetch,
    })
    await sdk.meters.get({ meterId: 'm' })
    expect(lastUrl()).toBe('https://eu.api.konghq.com/v3/openmeter/meters/m')
  })

  it('applies SDK auth after user beforeRequest hooks', async () => {
    mockMeter()
    const sdk = new OpenMeter({
      baseUrl: 'https://eu.api.konghq.com/v3',
      apiKey: 'real-key',
      fetch,
      hooks: {
        beforeRequest: [
          ({ request }) => {
            request.headers.set('Authorization', 'Bearer ATTACKER')
          },
        ],
      },
    })
    await sdk.meters.get({ meterId: 'm' })
    expect(lastAuth()).toBe('Bearer real-key')
  })
})

describe('SDK telemetry headers', () => {
  it('sets a default User-Agent header', async () => {
    mockMeter()
    const sdk = new OpenMeter({
      baseUrl: 'https://eu.api.konghq.com/v3',
      apiKey: 'k',
      fetch,
    })
    await sdk.meters.get({ meterId: 'm' })
    expect(lastUserAgent()).toBe(`openmeter-node/${SDK_VERSION}`)
  })

  it('does not overwrite a caller-provided User-Agent', async () => {
    mockMeter()
    const sdk = new OpenMeter({
      baseUrl: 'https://eu.api.konghq.com/v3',
      apiKey: 'k',
      headers: { 'User-Agent': 'custom-agent/1.0' },
      fetch,
    })
    await sdk.meters.get({ meterId: 'm' })
    expect(lastUserAgent()).toBe('custom-agent/1.0')
  })
})

describe('namespace composition', () => {
  it('memoizes namespace accessors', () => {
    const sdk = new OpenMeter({
      baseUrl: 'https://eu.api.konghq.com/v3',
      apiKey: 'k',
      fetch,
    })
    expect(sdk.meters).toBe(sdk.meters)
  })

  it('routes namespace calls through the root transport', async () => {
    mockMeter()
    const sdk = new OpenMeter({
      baseUrl: 'https://eu.api.konghq.com/v3',
      apiKey: 'k',
      fetch,
    })
    await sdk.meters.get({ meterId: 'm' })
    expect(lastUrl()).toBe('https://eu.api.konghq.com/v3/openmeter/meters/m')
    expect(lastAuth()).toBe('Bearer k')
  })
})
