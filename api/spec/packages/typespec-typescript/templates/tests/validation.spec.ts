import fetchMock from '@fetch-mock/vitest'
import { beforeEach, describe, expect, it } from 'vitest'
import { Client, funcs, ValidationError } from '../src/index.js'
import { toWire } from '../src/lib/wire.js'
import * as schemas from '../src/models/schemas.js'

const customerId = '01ARZ3NDEKTSV4RRFFQ69G5FAV'
const meterId = '01ARZ3NDEKTSV4RRFFQ69G5FAV'

const ingestedEvent = {
  event: {
    id: 'event-1',
    source: 'https://example.com/service',
    specversion: '1.0',
    type: 'api-request',
    subject: 'customer-1',
    time: '2024-01-01T00:00:00Z',
  },
  ingested_at: '2024-01-01T00:00:01Z',
  stored_at: '2024-01-01T00:00:02Z',
}

beforeEach(() => {
  fetchMock.mockReset()
})

function client(validate: boolean) {
  return new Client({
    baseUrl: 'https://eu.api.konghq.com/v3',
    apiKey: 'k',
    fetch: fetchMock.fetchHandler,
    validate,
  })
}

function json(body: unknown) {
  return {
    body,
    headers: { 'Content-Type': 'application/json' },
  }
}

describe('strict request validation uses the effective JSON payload', () => {
  it('omits explicit undefined object properties and record entries in toWire', () => {
    const wire = toWire(
      {
        timeZone: undefined,
        filters: {
          dimensions: {
            subject: undefined,
            customer_id: undefined,
          },
        },
      },
      schemas.meterQueryRequest,
    ) as Record<string, unknown>

    expect(Object.hasOwn(wire, 'time_zone')).toBe(false)
    expect(wire.filters).toEqual({ dimensions: {} })
  })

  it('sends a meter query with undefined optional dimension entries', async () => {
    let sentBody: unknown
    fetchMock.route('*', async ({ options }) => {
      sentBody = JSON.parse(options!.body as string)
      return json({ data: [] })
    })

    const result = await funcs.queryMeter(client(true), {
      meterId,
      body: {
        filters: {
          dimensions: {
            subject: undefined as never,
            customer_id: undefined as never,
            model: { eq: 'gpt-4o' },
          },
        },
      },
    })

    expect(result.ok).toBe(true)
    expect(sentBody).toEqual({
      filters: { dimensions: { model: { eq: 'gpt-4o' } } },
    })
  })

  it('rejects a genuinely invalid dimension filter before sending', async () => {
    fetchMock.route('*', json({ data: [] }))

    const result = await funcs.queryMeter(client(true), {
      meterId,
      body: {
        filters: {
          dimensions: { subject: 'not-a-filter-object' as never },
        },
      },
    })

    expect(result.ok).toBe(false)
    expect(result.error).toBeInstanceOf(ValidationError)
    expect(fetchMock.callHistory.calls()).toHaveLength(0)
  })

  it('keeps request validation disabled when validate is false', async () => {
    fetchMock.route('*', json({ data: [] }))

    const result = await funcs.queryMeter(client(false), {
      meterId,
      body: {
        filters: {
          dimensions: { subject: 'not-a-filter-object' as never },
        },
      },
    })

    expect(result.ok).toBe(true)
    expect(fetchMock.callHistory.calls()).toHaveLength(1)
  })

  it('materializes required defaults before strict wire validation', async () => {
    let sentBody: unknown
    fetchMock.route('*', async ({ options }) => {
      sentBody = JSON.parse(options!.body as string)
      return 204
    })

    const result = await funcs.ingestMeteringEvents(client(true), {
      id: 'event-with-default',
      source: 'https://example.com/service',
      type: 'api-request',
      subject: 'customer-1',
    })

    expect(result.ok).toBe(true)
    expect(sentBody).toMatchObject({ specversion: '1.0' })
  })
})

describe('strict sort validation follows the public and wire shapes', () => {
  const eventsPage = {
    data: [ingestedEvent],
    meta: { page: { size: 1, next: null, previous: null } },
  }

  it('accepts a valid public sort object and validates its encoded string', async () => {
    fetchMock.route('*', json(eventsPage))

    const result = await funcs.listMeteringEvents(client(true), {
      sort: { by: 'ingestedAt', order: 'desc' },
    })

    expect(result.ok).toBe(true)
    const query = new URL(fetchMock.callHistory.lastCall()!.url).searchParams
    expect(query.get('sort')).toBe('ingested_at desc')
  })

  it('rejects an invalid public sort before encoding or sending', async () => {
    fetchMock.route('*', json(eventsPage))

    const result = await funcs.listMeteringEvents(client(true), {
      sort: { by: 'ingestedAt', order: 'sideways' as never },
    })

    expect(result.ok).toBe(false)
    expect(result.error).toBeInstanceOf(ValidationError)
    expect(fetchMock.callHistory.calls()).toHaveLength(0)
  })

  it('keeps the existing sort behavior when validate is false', async () => {
    fetchMock.route('*', json(eventsPage))

    const result = await funcs.listMeteringEvents(client(false), {
      sort: { by: 'ingestedAt', order: 'sideways' as never },
    })

    expect(result.ok).toBe(true)
    const query = new URL(fetchMock.callHistory.lastCall()!.url).searchParams
    expect(query.get('sort')).toBe('ingested_at')
  })
})

describe('strict path validation uses generated path schemas', () => {
  it('rejects an invalid constrained path value before sending', async () => {
    fetchMock.route('*', json({}))

    const result = await funcs.getMeter(client(true), {
      meterId: 'not-a-ulid',
    })

    expect(result.ok).toBe(false)
    expect(result.error).toBeInstanceOf(ValidationError)
    expect(fetchMock.callHistory.calls()).toHaveLength(0)
  })

  it('keeps path validation disabled when validate is false', async () => {
    fetchMock.route('*', json({}))

    const result = await funcs.getMeter(client(false), {
      meterId: 'not-a-ulid',
    })

    expect(result.ok).toBe(true)
    expect(fetchMock.callHistory.calls()).toHaveLength(1)
    expect(fetchMock.callHistory.lastCall()!.url).toContain(
      '/openmeter/meters/not-a-ulid',
    )
  })
})

describe('nullable cursor response validation', () => {
  it('accepts an events page with a null previous cursor and returns its events', async () => {
    fetchMock.route(
      '*',
      json({
        data: [ingestedEvent],
        meta: { page: { size: 1, next: null, previous: null } },
      }),
    )

    const result = await funcs.listMeteringEvents(client(true), {})

    expect(result.ok).toBe(true)
    expect(result.value?.data).toHaveLength(1)
    expect(result.value?.data[0]?.event.id).toBe('event-1')
    expect(result.value?.meta.page.previous).toBeNull()
  })

  it('accepts an empty credit transaction page with null cursors', async () => {
    fetchMock.route(
      '*',
      json({
        data: [],
        meta: { page: { size: 100, next: null, previous: null } },
      }),
    )

    const result = await funcs.listCreditTransactions(client(true), {
      customerId,
    })

    expect(result.ok).toBe(true)
    expect(result.value?.data).toEqual([])
    expect(result.value?.meta.page.next).toBeNull()
    expect(result.value?.meta.page.previous).toBeNull()
  })

  it('rejects a genuinely invalid cursor value', async () => {
    fetchMock.route(
      '*',
      json({
        data: [ingestedEvent],
        meta: { page: { size: 1, next: null, previous: 42 } },
      }),
    )

    const result = await funcs.listMeteringEvents(client(true), {})

    expect(result.ok).toBe(false)
    expect(result.error).toBeInstanceOf(ValidationError)
  })

  it('rejects a response missing a required field that has a request default', async () => {
    const { specversion: _specversion, ...eventWithoutSpecversion } =
      ingestedEvent.event
    fetchMock.route(
      '*',
      json({
        data: [{ ...ingestedEvent, event: eventWithoutSpecversion }],
        meta: { page: { size: 1, next: null, previous: null } },
      }),
    )

    const result = await funcs.listMeteringEvents(client(true), {})

    expect(result.ok).toBe(false)
    expect(result.error).toBeInstanceOf(ValidationError)
  })

  it('keeps response validation disabled when validate is false', async () => {
    fetchMock.route(
      '*',
      json({
        data: [ingestedEvent],
        meta: { page: { size: 1, next: null, previous: 42 } },
      }),
    )

    const result = await funcs.listMeteringEvents(client(false), {})

    expect(result.ok).toBe(true)
    expect(result.value?.data).toHaveLength(1)
    expect(result.value?.meta.page.previous).toBe(42)
  })
})
