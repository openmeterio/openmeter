import fetchMock from '@fetch-mock/vitest'
import { beforeEach, describe, expect, it } from 'vitest'
import { z } from 'zod'
import { Client, funcs, ValidationError } from '../src/index.js'
import * as schemas from '../src/models/schemas.js'
import { DepthLimitExceededError, fromWire, toWire } from '../src/lib/wire.js'

beforeEach(() => {
  fetchMock.mockReset()
})

function client() {
  return new Client({
    baseUrl: 'https://eu.api.konghq.com/v3',
    apiKey: 'k',
    fetch: fetchMock.fetchHandler,
  })
}

describe('wire mapper (toWire/fromWire over real schemas)', () => {
  it('preserves record keys and renames value fields (governance)', () => {
    const wire = { features: { my_user_feature: { has_access: true } } }
    const camel = fromWire(wire, schemas.governanceQueryResult) as any
    expect(Object.keys(camel.features)[0]).toBe('my_user_feature')
    expect(camel.features.my_user_feature.hasAccess).toBe(true)

    const back = toWire(
      { features: { my_user_feature: { hasAccess: true } } },
      schemas.governanceQueryResult,
    ) as any
    expect(Object.keys(back.features)[0]).toBe('my_user_feature')
    expect(back.features.my_user_feature.has_access).toBe(true)
  })

  it('preserves a record entry keyed "__proto__" as a visible own property', () => {
    // A literal object expression treats `__proto__` as a prototype setter,
    // not a data key — build it the way a real HTTP response body arrives
    // (JSON.parse always yields `__proto__` as an own enumerable property).
    const wire = JSON.parse('{"features":{"__proto__":{"has_access":true}}}')
    const camel = fromWire(wire, schemas.governanceQueryResult) as any
    expect(Object.keys(camel.features)).toEqual(['__proto__'])
    expect(camel.features.__proto__.hasAccess).toBe(true)
    expect(Object.getPrototypeOf({})).toBe(Object.prototype)
  })

  it('treats a data key of "constructor" as an unknown field, not a shape hit', () => {
    const wire = JSON.parse('{"id":"x","constructor":"evil"}')
    const camel = fromWire(wire, schemas.taxCode) as any
    expect(camel.constructor).toBeUndefined()
    expect(Object.keys(camel)).not.toContain('constructor')
  })

  it('throws a typed DepthLimitExceededError instead of a raw stack overflow on a deeply nested self-referential filter', () => {
    let deep: unknown = { eq: 'leaf' }
    for (let i = 0; i < 1000; i++) {
      deep = { and: [deep] }
    }
    expect(() => fromWire(deep, schemas.queryFilterString)).toThrow(
      DepthLimitExceededError,
    )
  })

  it('walks a moderately nested filter without hitting the depth limit', () => {
    let nested: unknown = { eq: 'leaf' }
    for (let i = 0; i < 20; i++) {
      nested = { and: [nested] }
    }
    const wire = { and: [nested] }
    expect(() => fromWire(wire, schemas.queryFilterString)).not.toThrow()
  })

  it('handles a multi-word discriminator (collection_method)', () => {
    const wire = { collection_method: 'charge_automatically' }
    const camel = fromWire(wire, schemas.workflowPaymentSettings) as any
    expect(camel.collectionMethod).toBe('charge_automatically')

    const back = toWire(
      { collectionMethod: 'charge_automatically' },
      schemas.workflowPaymentSettings,
    ) as any
    expect(back.collection_method).toBe('charge_automatically')
  })

  it('preserves meter dimension record keys', () => {
    const wire = { dimensions: { my_dim_key: { eq: 'x' } } }
    const camel = fromWire(wire, schemas.meterQueryFilters) as any
    expect('my_dim_key' in camel.dimensions).toBe(true)
    expect(camel.dimensions.my_dim_key.eq).toBe('x')
  })

  it('terminates on a cyclic filter (and/or self-reference)', () => {
    const wire = {
      dimensions: {
        my_dim: { eq: 'a', and: [{ eq: 'b' }, { or: [{ eq: 'c' }] }] },
      },
    }
    const camel = fromWire(wire, schemas.meterQueryFilters) as any
    expect(camel.dimensions.my_dim.and[1].or[0].eq).toBe('c')
  })

  it('preserves arbitrary keys inside a record of unknown (event.data)', () => {
    const wire = { type: 'x', source: 's', data: { user_set_key: { a_b: 1 } } }
    const camel = fromWire(wire, schemas.event) as any
    expect(camel.data.user_set_key).toEqual({ a_b: 1 })
  })

  it('walks every element of a single-or-batch (T | T[]) body', () => {
    // Mirrors the ingest body shape `EventInput | EventInput[]`. A batch must
    // walk each element, not pass it through untransformed (the array branch
    // must resolve the array variant's element schema from the union).
    const single = z.object({ eventType: z.string() })
    const body = z.union([single, z.array(single)])

    const batch = toWire(
      [{ eventType: 'a' }, { eventType: 'b' }],
      body,
    ) as any[]
    expect(batch).toHaveLength(2)
    for (const e of batch) {
      expect('event_type' in e).toBe(true)
      expect('eventType' in e).toBe(false)
    }

    const one = toWire({ eventType: 'a' }, body) as any
    expect(one.event_type).toBe('a')
  })
})

describe('end-to-end wire mapping through a func', () => {
  it('sends a snake body and returns a camelCase response (createMeter)', async () => {
    let sentBody: any
    fetchMock.route('*', async ({ options }) => {
      sentBody = JSON.parse(options!.body as string)
      return {
        body: {
          id: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
          name: 'API calls',
          key: 'api_calls',
          aggregation: 'count',
          event_type: 'api-request',
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
        headers: { 'Content-Type': 'application/json' },
      }
    })

    const result = await funcs.createMeter(client(), {
      name: 'API calls',
      key: 'api_calls',
      aggregation: 'count',
      eventType: 'api-request',
      valueProperty: '$.value',
    })

    // request body is snake on the wire
    expect(sentBody.event_type).toBe('api-request')
    expect(sentBody.value_property).toBe('$.value')
    expect('eventType' in sentBody).toBe(false)

    // response is camelCase
    expect(result.ok).toBe(true)
    expect(result.value).toMatchObject({
      eventType: 'api-request',
      createdAt: '2024-01-01T00:00:00Z',
    })
  })
})

describe('wire walker edge cases', () => {
  it('passes a scalar/null/array-of-scalars through unchanged', () => {
    expect(toWire(5, z.number())).toBe(5)
    expect(toWire(null, z.string().nullable())).toBe(null)
    expect(toWire([1, 2], z.array(z.number()))).toEqual([1, 2])
  })

  it('recurses a record whose value is an array or union of models', () => {
    const arrModel = z.record(
      z.string(),
      z.array(z.object({ fooBar: z.string() })),
    )
    expect(toWire({ user_k: [{ fooBar: 'x' }] }, arrModel)).toEqual({
      user_k: [{ foo_bar: 'x' }],
    })

    const unionModel = z.record(
      z.string(),
      z.union([
        z.object({ fooBar: z.string() }),
        z.object({ bazQux: z.number() }),
      ]),
    )
    expect(toWire({ user_k: { fooBar: 'x' } }, unionModel)).toEqual({
      user_k: { foo_bar: 'x' },
    })
  })

  it('leaves a record of scalars untouched (keys and values)', () => {
    const scalarMap = z.record(z.string(), z.string())
    expect(toWire({ a_b: 'v', cD: 'w' }, scalarMap)).toEqual({
      a_b: 'v',
      cD: 'w',
    })
  })

  it('renames the object variant of a scalar-or-object union', () => {
    // The realistic non-discriminated shape (a filter field: string | { op }).
    // The codegen gate guarantees at most one object variant, so it is picked
    // unambiguously; scalar data flows through the scalar branch unchanged.
    const union = z.union([z.string(), z.object({ fooBar: z.string() })])
    expect(toWire({ fooBar: 'x' }, union)).toEqual({ foo_bar: 'x' })
    expect(toWire('plain', union)).toBe('plain')
  })

  it('fails closed on a discriminated union with an unknown discriminator value', () => {
    const union = z.discriminatedUnion('kind', [
      z.object({ kind: z.literal('a'), fooBar: z.string() }),
      z.object({ kind: z.literal('b'), bazQux: z.number() }),
    ])
    // no variant has kind === 'z' → returned unchanged
    expect(toWire({ kind: 'z', fooBar: 'x' }, union)).toEqual({
      kind: 'z',
      fooBar: 'x',
    })
  })

  it('reuses the memoized variant map across calls on the same union', () => {
    const union = z.discriminatedUnion('kind', [
      z.object({ kind: z.literal('a'), fooBar: z.string() }),
      z.object({ kind: z.literal('b'), bazQux: z.number() }),
    ])
    // First call builds the map; second hits the cache. Both dispatch correctly.
    expect(toWire({ kind: 'a', fooBar: 'x' }, union)).toEqual({
      kind: 'a',
      foo_bar: 'x',
    })
    expect(toWire({ kind: 'b', bazQux: 2 }, union)).toEqual({
      kind: 'b',
      baz_qux: 2,
    })
  })

  it('walks array data against a union schema with an array variant', () => {
    // exercises arrayElement resolving the array option of a union
    const union = z.union([
      z.object({ fooBar: z.string() }),
      z.array(z.object({ fooBar: z.string() })),
    ])
    expect(toWire([{ fooBar: 'x' }], union)).toEqual([{ foo_bar: 'x' }])
  })

  it('passes array data through when the union has no array variant', () => {
    // arrayElement returns undefined → elements walked with no schema → unchanged
    const union = z.union([
      z.object({ fooBar: z.string() }),
      z.object({ bazQux: z.number() }),
    ])
    expect(toWire([{ fooBar: 'x' }], union)).toEqual([{ fooBar: 'x' }])
  })

  it('keeps a record whose value schema is an unknown/any (event.data-like)', () => {
    const rec = z.record(z.string(), z.unknown())
    expect(toWire({ user_k: { aB: 1 } }, rec)).toEqual({ user_k: { aB: 1 } })
  })
})

describe('optional schema validation (validate option)', () => {
  const goodMeter = {
    id: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
    name: 'API calls',
    key: 'api_calls',
    aggregation: 'count',
    event_type: 'api-request',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  }
  const goodCreate = {
    name: 'API calls',
    key: 'api_calls',
    aggregation: 'count' as const,
    eventType: 'api-request',
    valueProperty: '$.value',
  }

  function validatingClient() {
    return new Client({
      baseUrl: 'https://eu.api.konghq.com/v3',
      apiKey: 'k',
      fetch: fetchMock.fetchHandler,
      validate: true,
    })
  }

  it('passes a valid request and response when validate is on', async () => {
    fetchMock.route('*', {
      body: goodMeter,
      headers: { 'Content-Type': 'application/json' },
    })
    const result = await funcs.createMeter(validatingClient(), goodCreate)
    expect(result.ok).toBe(true)
    expect(result.value).toMatchObject({ eventType: 'api-request' })
  })

  it('rejects a request body that fails its schema', async () => {
    fetchMock.route('*', {
      body: goodMeter,
      headers: { 'Content-Type': 'application/json' },
    })
    // name is required; omit it
    const result = await funcs.createMeter(validatingClient(), {
      ...goodCreate,
      name: undefined as unknown as string,
    })
    expect(result.ok).toBe(false)
    expect(result.error).toBeInstanceOf(ValidationError)
  })

  it('rejects an enum-drift response (the reason validation is off by default)', async () => {
    fetchMock.route('*', {
      // server sends an aggregation value this SDK version does not know
      body: { ...goodMeter, aggregation: 'p99' },
      headers: { 'Content-Type': 'application/json' },
    })
    const result = await funcs.getMeter(validatingClient(), { meterId: 'm' })
    expect(result.ok).toBe(false)
    expect(result.error).toBeInstanceOf(ValidationError)
  })

  it('does NOT reject the same enum-drift response when validate is off', async () => {
    fetchMock.route('*', {
      body: { ...goodMeter, aggregation: 'p99' },
      headers: { 'Content-Type': 'application/json' },
    })
    const result = await funcs.getMeter(client(), { meterId: 'm' })
    expect(result.ok).toBe(true)
  })

  it('rejects a bad body on a void/no-JSON-response op as Result.error, not a throw', async () => {
    // ingestMeteringEvents has a request body but a void response; body validation
    // must run inside request() so a failure becomes Result.error here too.
    fetchMock.route('*', { status: 204 })
    const result = await funcs.ingestMeteringEvents(
      validatingClient(),
      {} as never,
    )
    expect(result.ok).toBe(false)
    expect(result.error).toBeInstanceOf(ValidationError)
  })
})

describe('generated wire schemas (snake_case, strict)', () => {
  const ok = (s: { safeParse(v: unknown): { success: boolean } }, v: unknown) =>
    s.safeParse(v).success

  const meterWireData = {
    id: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
    name: 'n',
    key: 'k',
    aggregation: 'count',
    event_type: 'e',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  }

  it('a closed model wire schema is snake-keyed and strict', () => {
    expect(ok(schemas.meterWire, meterWireData)).toBe(true)
    // leaked camelCase key rejected
    expect(ok(schemas.meterWire, { ...meterWireData, eventType: 'x' })).toBe(
      false,
    )
    // unknown/extra wire key rejected
    expect(ok(schemas.meterWire, { ...meterWireData, new_field: 1 })).toBe(
      false,
    )
  })

  it('a record value model wire schema preserves user keys, snakeifies fields', () => {
    expect(ok(schemas.governanceFeatureAccessWire, { has_access: true })).toBe(
      true,
    )
    expect(ok(schemas.governanceFeatureAccessWire, { hasAccess: true })).toBe(
      false,
    )
  })

  it('a discriminated union wire schema uses the snake discriminator', () => {
    expect(
      ok(schemas.workflowPaymentSettingsWire, {
        collection_method: 'charge_automatically',
      }),
    ).toBe(true)
  })

  it('an open (record-spread) wire schema accepts extra keys (not strict)', () => {
    // baseErrorWire is emitsAsIntersection — strict would defeat the record arm
    // that exists to accept additional members. Must stay permissive.
    expect(
      ok(schemas.baseErrorWire, {
        type: 't',
        status: 400,
        title: 'x',
        detail: 'd',
        instance: '/i',
        anything_extra: 1,
      }),
    ).toBe(true)
  })

  it('a cyclic filter wire schema terminates and accepts nested data', () => {
    expect(
      ok(schemas.meterQueryFiltersWire, {
        dimensions: { my_dim: { eq: 'a', and: [{ eq: 'b' }] } },
      }),
    ).toBe(true)
  })
})
