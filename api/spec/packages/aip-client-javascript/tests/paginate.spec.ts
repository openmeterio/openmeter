import fetchMock from '@fetch-mock/vitest'
import { beforeEach, describe, expect, it } from 'vitest'
import { OpenMeter, type IngestedEvent, type Meter } from '../src/index.js'
import {
  PaginationLimitExceededError,
  paginateCursor,
  paginatePages,
} from '../src/lib/paginate.js'
import type { Result } from '../src/lib/types.js'

beforeEach(() => {
  fetchMock.mockReset()
})

function client() {
  return new OpenMeter({
    baseUrl: 'https://eu.api.konghq.com/v3',
    apiKey: 'k',
    fetch: fetchMock.fetchHandler,
  })
}

function meterFixture(key: string) {
  return {
    id: `01ARZ3NDEKTSV4RRFFQ69G5${key.toUpperCase().padStart(4, '0')}`,
    name: key,
    key,
    aggregation: 'count',
    event_type: 'api-request',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  }
}

function eventFixture(id: string) {
  return {
    event: { id, source: 'test', specversion: '1.0', type: 'test.event' },
    ingested_at: '2024-01-01T00:00:00Z',
    stored_at: '2024-01-01T00:00:00Z',
  }
}

function jsonRoute(body: unknown) {
  return { body, headers: { 'Content-Type': 'application/json' } }
}

describe('paginatePages (page-number pagination)', () => {
  it('iterates every item across 3 pages via client.meters.listAll', async () => {
    // total 5 across pages of size 2, 2, 1 — the last page is short, which is
    // what actually stops the iteration (see the exact-total test below for
    // the other stop condition).
    const pageKeys = [['a', 'b'], ['c', 'd'], ['e']]
    fetchMock.route('*', (callLog) => {
      const number = Number(callLog.queryParams?.get('page[number]') ?? '1')
      const keys = pageKeys[number - 1] ?? []
      return jsonRoute({
        data: keys.map(meterFixture),
        meta: { page: { number, size: 2, total: 5 } },
      })
    })

    const keys: string[] = []
    for await (const meter of client().meters.listAll()) {
      keys.push(meter.key)
    }

    expect(keys).toEqual(['a', 'b', 'c', 'd', 'e'])
    expect(fetchMock.callHistory.calls()).toHaveLength(3)
  })

  it('stops without an extra request when the final page exactly fills total', async () => {
    fetchMock.route('*', (callLog) => {
      const number = Number(callLog.queryParams?.get('page[number]') ?? '1')
      const keys = number === 1 ? ['a', 'b'] : []
      return jsonRoute({
        data: keys.map(meterFixture),
        meta: { page: { number, size: 2, total: 2 } },
      })
    })

    const keys: string[] = []
    for await (const meter of client().meters.listAll()) {
      keys.push(meter.key)
    }

    expect(keys).toEqual(['a', 'b'])
    expect(fetchMock.callHistory.calls()).toHaveLength(1)
  })

  it('yields nothing and fetches once for an empty first page', async () => {
    fetchMock.route(
      '*',
      jsonRoute({
        data: [],
        meta: { page: { number: 1, size: 10, total: 0 } },
      }),
    )

    const keys: string[] = []
    for await (const meter of client().meters.listAll()) {
      keys.push(meter.key)
    }

    expect(keys).toEqual([])
    expect(fetchMock.callHistory.calls()).toHaveLength(1)
  })

  it('fires no further requests once the consumer breaks early', async () => {
    const pageKeys = [
      ['a', 'b'],
      ['c', 'd'],
    ]
    fetchMock.route('*', (callLog) => {
      const number = Number(callLog.queryParams?.get('page[number]') ?? '1')
      return jsonRoute({
        data: (pageKeys[number - 1] ?? []).map(meterFixture),
        meta: { page: { number, size: 2, total: 4 } },
      })
    })

    for await (const meter of client().meters.listAll()) {
      expect(meter.key).toBe('a')
      break
    }

    expect(fetchMock.callHistory.calls()).toHaveLength(1)
  })

  it('sends the caller-supplied filter and sort on every page, advancing only page.number', async () => {
    fetchMock.route('*', (callLog) => {
      const number = Number(callLog.queryParams?.get('page[number]') ?? '1')
      const keys = number < 2 ? ['a', 'b'] : ['c']
      return jsonRoute({
        data: keys.map(meterFixture),
        meta: { page: { number, size: 2, total: 3 } },
      })
    })

    const items: Meter[] = []
    for await (const meter of client().meters.listAll({
      filter: { key: { eq: 'api' } },
      sort: { by: 'createdAt', order: 'desc' },
    })) {
      items.push(meter)
    }

    expect(items).toHaveLength(3)
    const calls = fetchMock.callHistory.calls()
    expect(calls).toHaveLength(2)
    for (const [i, call] of calls.entries()) {
      const q = new URL(call.url).searchParams
      expect(q.get('filter[key][eq]')).toBe('api')
      expect(q.get('sort')).toBe('created_at desc')
      // The first request is whatever the caller supplied — no page.number at
      // all, left to the server's own default — only later requests carry an
      // explicit page.number, advanced from the previous response.
      expect(q.get('page[number]')).toBe(i === 0 ? null : String(i + 1))
    }
  })

  it('propagates the AbortSignal to every page fetch, and aborting stops iteration', async () => {
    const controller = new AbortController()
    // ky composes the caller's signal into its own dependent AbortSignal
    // before it reaches fetch (its `.aborted` state tracks the original, but
    // it is not the same object), so propagation is verified behaviorally —
    // a signal reaches every dispatched request, and aborting mid-iteration
    // stops it — rather than by reference identity.
    const signals: (AbortSignal | undefined)[] = []
    fetchMock.route('*', (callLog) => {
      signals.push(callLog.signal)
      const number = Number(callLog.queryParams?.get('page[number]') ?? '1')
      return jsonRoute({
        data: [meterFixture(`m${number}`)],
        meta: { page: { number, size: 1, total: 5 } },
      })
    })

    const keys: string[] = []
    await expect(async () => {
      for await (const meter of client().meters.listAll(undefined, {
        signal: controller.signal,
      })) {
        keys.push(meter.key)
        controller.abort()
      }
    }).rejects.toThrow()

    // The abort races with the second page's already-dispatched request: the
    // mock still resolves a response for it (this route intercepts before
    // ky/fetch's own abort check settles the promise), but the abort still
    // wins — no page-2 item is yielded, and the iterable rejects instead of
    // completing normally.
    expect(keys).toEqual(['m1'])
    expect(signals.length).toBeGreaterThanOrEqual(1)
    expect(signals.every((signal) => signal !== undefined)).toBe(true)
  })

  it('throws PaginationLimitExceededError instead of looping forever on a server that never signals the end', async () => {
    let calls = 0
    const fetchPage = async (): Promise<
      Result<{
        data: unknown[]
        meta: { page: { number: number; size: number; total: number } }
      }>
    > => {
      calls++
      return {
        ok: true,
        value: {
          data: [{}],
          meta: {
            page: { number: calls, size: 1, total: Number.MAX_SAFE_INTEGER },
          },
        },
      }
    }

    await expect(async () => {
      for await (const _ of paginatePages(fetchPage, {})) {
        // drain
      }
    }).rejects.toThrow(PaginationLimitExceededError)
    expect(calls).toBe(10_000)
  })
})

describe('paginateCursor (cursor pagination)', () => {
  it('iterates every item across 3 pages via client.events.listAll', async () => {
    const pages: Record<string, { ids: string[]; next?: string }> = {
      '': { ids: ['1', '2'], next: 'cursor-2' },
      'cursor-2': { ids: ['3', '4'], next: 'cursor-3' },
      'cursor-3': { ids: ['5'], next: undefined },
    }
    fetchMock.route('*', (callLog) => {
      const after = callLog.queryParams?.get('page[after]') ?? ''
      const page = pages[after]!
      return jsonRoute({
        data: page.ids.map(eventFixture),
        meta: { page: { next: page.next ?? null } },
      })
    })

    const ids: string[] = []
    for await (const event of client().events.listAll()) {
      ids.push(event.event.id)
    }

    expect(ids).toEqual(['1', '2', '3', '4', '5'])
    expect(fetchMock.callHistory.calls()).toHaveLength(3)
  })

  it('stops after one fetch when the first page has no next cursor', async () => {
    fetchMock.route(
      '*',
      jsonRoute({
        data: [eventFixture('1')],
        meta: { page: { next: null } },
      }),
    )

    const ids: string[] = []
    for await (const event of client().events.listAll()) {
      ids.push(event.event.id)
    }

    expect(ids).toEqual(['1'])
    expect(fetchMock.callHistory.calls()).toHaveLength(1)
  })

  it('fires no further requests once the consumer breaks early', async () => {
    const pages: Record<string, { ids: string[]; next?: string }> = {
      '': { ids: ['1', '2'], next: 'cursor-2' },
      'cursor-2': { ids: ['3'], next: undefined },
    }
    fetchMock.route('*', (callLog) => {
      const after = callLog.queryParams?.get('page[after]') ?? ''
      const page = pages[after]!
      return jsonRoute({
        data: page.ids.map(eventFixture),
        meta: { page: { next: page.next ?? null } },
      })
    })

    for await (const event of client().events.listAll()) {
      expect(event.event.id).toBe('1')
      break
    }

    expect(fetchMock.callHistory.calls()).toHaveLength(1)
  })

  it('feeds meta.page.next back as page.after verbatim (an opaque token, not a URI to fetch)', async () => {
    const pages: Record<string, { ids: string[]; next?: string }> = {
      '': { ids: ['1'], next: 'opaque-token-not-a-url' },
      'opaque-token-not-a-url': { ids: ['2'], next: undefined },
    }
    fetchMock.route('*', (callLog) => {
      const after = callLog.queryParams?.get('page[after]') ?? ''
      const page = pages[after]!
      return jsonRoute({
        data: page.ids.map(eventFixture),
        meta: { page: { next: page.next ?? null } },
      })
    })

    const ids: string[] = []
    for await (const event of client().events.listAll()) {
      ids.push(event.event.id)
    }

    expect(ids).toEqual(['1', '2'])
  })

  it('sends the caller-supplied filter on every page, advancing only page.after', async () => {
    const pages: Record<string, { ids: string[]; next?: string }> = {
      '': { ids: ['1'], next: 'cursor-2' },
      'cursor-2': { ids: ['2'], next: undefined },
    }
    fetchMock.route('*', (callLog) => {
      const after = callLog.queryParams?.get('page[after]') ?? ''
      const page = pages[after]!
      return jsonRoute({
        data: page.ids.map(eventFixture),
        meta: { page: { next: page.next ?? null } },
      })
    })

    for await (const _ of client().events.listAll({
      filter: { subject: { eq: 'customer-1' } },
    })) {
      // drain
    }

    const calls = fetchMock.callHistory.calls()
    expect(calls).toHaveLength(2)
    for (const call of calls) {
      expect(new URL(call.url).searchParams.get('filter[subject][eq]')).toBe(
        'customer-1',
      )
    }
  })

  it('throws PaginationLimitExceededError instead of looping forever on a server that never stops returning a next cursor', async () => {
    let calls = 0
    const fetchPage = async (): Promise<
      Result<{ data: unknown[]; meta: { page: { next?: string } } }>
    > => {
      calls++
      return {
        ok: true,
        value: { data: [{}], meta: { page: { next: 'again' } } },
      }
    }

    await expect(async () => {
      for await (const _ of paginateCursor(fetchPage, {})) {
        // drain
      }
    }).rejects.toThrow(PaginationLimitExceededError)
    expect(calls).toBe(10_000)
  })
})

describe('AsyncIterable<Item> type inference', () => {
  // Compile-time proof that the facade companions infer the ITEM type, not
  // the page envelope — `listAll`'s return type must be exactly
  // `AsyncIterable<Meter>`/`AsyncIterable<IngestedEvent>`, with no cast at the
  // call site required to get there.
  type ListAllItem<T> = T extends AsyncIterable<infer Item> ? Item : never
  type _MetersListAllYieldsMeter = [
    ListAllItem<ReturnType<OpenMeter['meters']['listAll']>>,
  ] extends [Meter]
    ? [Meter] extends [ListAllItem<ReturnType<OpenMeter['meters']['listAll']>>]
      ? true
      : { __error: 'meters.listAll yields something narrower than Meter' }
    : { __error: 'meters.listAll does not yield Meter' }
  const _metersListAllYieldsMeter: _MetersListAllYieldsMeter = true

  type _EventsListAllYieldsIngestedEvent = [
    ListAllItem<ReturnType<OpenMeter['events']['listAll']>>,
  ] extends [IngestedEvent]
    ? [IngestedEvent] extends [
        ListAllItem<ReturnType<OpenMeter['events']['listAll']>>,
      ]
      ? true
      : {
          __error: 'events.listAll yields something narrower than IngestedEvent'
        }
    : { __error: 'events.listAll does not yield IngestedEvent' }
  const _eventsListAllYieldsIngestedEvent: _EventsListAllYieldsIngestedEvent = true

  it('type-checks the item-level inference probes above', () => {
    expect(_metersListAllYieldsMeter).toBe(true)
    expect(_eventsListAllYieldsIngestedEvent).toBe(true)
  })
})
