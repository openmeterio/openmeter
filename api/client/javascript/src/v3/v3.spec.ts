import fetchMock from '@fetch-mock/vitest'
import { beforeEach, describe, expect, it } from 'vitest'
import { isHTTPError, OpenMeter } from '../client/index.js'

// Smoke test for the v3 compatibility shim. Mirrors the runtime-validation
// intent of the generated v3 SDK's client-runtime-test.mjs: it exercises base
// URL composition, request shaping (method/path/query/body), snake_case body
// passthrough (Option A — no field renaming), and error throwing. The transport
// is stubbed via fetch-mock; no network is touched.

interface Context {
  baseUrl: string
  client: OpenMeter
}

describe('v3 shim', () => {
  beforeEach<Context>((ctx) => {
    fetchMock.mockReset()
    const baseUrl = 'http://openmeter-mock.local'
    ctx.baseUrl = baseUrl
    ctx.client = new OpenMeter({
      apiKey: 'test-key',
      baseUrl,
      fetch: fetchMock.fetchHandler,
    })
  })

  it<Context>('create plan: POST <baseUrl>/api/v3/openmeter/plans with snake_case body', async ({
    baseUrl,
    client,
    task,
  }) => {
    const route = `${baseUrl}/api/v3/openmeter/plans`
    const created = { id: '01J0000000000000000000PLAN', key: 'starter' }
    fetchMock.route(
      route,
      { body: created, status: 201 },
      { method: 'POST', name: task.name },
    )

    const resp = await client.v3.plans.create({
      billing_cadence: 'P1M',
      currency: 'USD',
      key: 'starter',
      name: 'Starter',
      phases: [],
    })

    // Response body deserialized and returned.
    expect(resp).toMatchObject({ id: created.id, key: 'starter' })
    expect(fetchMock.callHistory.done(task.name)).toBeTruthy()

    // The base URL is composed as <baseUrl>/api/v3 and the snake_case body is
    // sent verbatim (no camelCase renaming).
    const call = fetchMock.callHistory.calls()[0]
    expect(call?.url).toBe(route)
    const sent = JSON.parse(String(call?.options?.body))
    expect(sent.billing_cadence).toBe('P1M')
    expect(sent.key).toBe('starter')
  })

  it<Context>('create addon: POST <baseUrl>/api/v3/openmeter/addons with snake_case body', async ({
    baseUrl,
    client,
    task,
  }) => {
    const route = `${baseUrl}/api/v3/openmeter/addons`
    const created = { id: '01J0000000000000000000ADDON', key: 'extra' }
    fetchMock.route(
      route,
      { body: created, status: 201 },
      { method: 'POST', name: task.name },
    )

    const resp = await client.v3.addons.create({
      currency: 'USD',
      instance_type: 'single',
      key: 'extra',
      name: 'Extra',
      rate_cards: [],
    })

    expect(resp).toMatchObject({ id: created.id, key: 'extra' })
    expect(fetchMock.callHistory.done(task.name)).toBeTruthy()

    const call = fetchMock.callHistory.calls()[0]
    expect(call?.url).toBe(route)
    const sent = JSON.parse(String(call?.options?.body))
    // snake_case body field passed through verbatim (Option A).
    expect(sent.instance_type).toBe('single')
    expect(sent.key).toBe('extra')
  })

  it<Context>('list addons: GET /openmeter/addons with pagination query', async ({
    baseUrl,
    client,
    task,
  }) => {
    const route = `${baseUrl}/api/v3/openmeter/addons`
    const page = { data: [], meta: { page: 1, page_size: 20, total_count: 0 } }
    // `begin:` matcher so the deepObject query string doesn't break the match.
    fetchMock.route(
      `begin:${route}`,
      { body: page, status: 200 },
      { method: 'GET', name: task.name },
    )

    const resp = await client.v3.addons.list({
      page: { number: 1, size: 20 },
    })

    expect(resp).toEqual(page)
    const calledUrl = decodeURIComponent(
      String(fetchMock.callHistory.calls()[0]?.url),
    )
    expect(calledUrl).toContain('page[size]=20')
  })

  it<Context>('list meters: deepObject page[...] pagination query', async ({
    baseUrl,
    client,
    task,
  }) => {
    const route = `${baseUrl}/api/v3/openmeter/meters`
    const page = { data: [], meta: { page: 2, page_size: 50, total_count: 0 } }
    // `begin:` matcher so the deepObject query string doesn't break the match.
    fetchMock.route(
      `begin:${route}`,
      { body: page, status: 200 },
      { method: 'GET', name: task.name },
    )

    const resp = await client.v3.meters.list({
      page: { number: 2, size: 50 },
    })

    expect(resp).toEqual(page)
    const calledUrl = decodeURIComponent(
      String(fetchMock.callHistory.calls()[0]?.url),
    )
    expect(calledUrl).toContain('page[size]=50')
    expect(calledUrl).toContain('page[number]=2')
  })

  it<Context>('error: 4xx throws HTTPError', async ({
    baseUrl,
    client,
    task,
  }) => {
    const route = `${baseUrl}/api/v3/openmeter/plans/missing`
    fetchMock.route(
      route,
      {
        body: { status: 404, title: 'Not Found', type: 'about:blank' },
        headers: { 'Content-Type': 'application/problem+json' },
        status: 404,
      },
      { method: 'GET', name: task.name },
    )

    await expect(client.v3.plans.get('missing')).rejects.toSatisfy(
      (err: unknown) => isHTTPError(err) && err.status === 404,
    )
  })

  it<Context>('update plan: PUT /openmeter/plans/{planId} with snake_case body', async ({
    baseUrl,
    client,
    task,
  }) => {
    const route = `${baseUrl}/api/v3/openmeter/plans/01PLAN`
    fetchMock.route(
      route,
      { body: { id: '01PLAN', key: 'starter' }, status: 200 },
      { method: 'PUT', name: task.name },
    )

    const resp = await client.v3.plans.update('01PLAN', {
      name: 'Starter',
      phases: [],
      pro_rating_enabled: true,
    })

    expect(resp).toMatchObject({ id: '01PLAN' })
    const sent = JSON.parse(
      String(fetchMock.callHistory.calls()[0]?.options?.body),
    )
    // snake_case body field passed through verbatim (Option A).
    expect(sent.pro_rating_enabled).toBe(true)
  })

  it<Context>('update feature: PATCH /openmeter/features/{featureId}', async ({
    baseUrl,
    client,
    task,
  }) => {
    const route = `${baseUrl}/api/v3/openmeter/features/01FEAT`
    fetchMock.route(
      route,
      { body: { id: '01FEAT' }, status: 200 },
      { method: 'PATCH', name: task.name },
    )

    const resp = await client.v3.features.update('01FEAT', {})

    expect(resp).toMatchObject({ id: '01FEAT' })
    expect(fetchMock.callHistory.done(task.name)).toBeTruthy()
  })

  it<Context>('delete meter: DELETE /openmeter/meters/{meterId} (204)', async ({
    baseUrl,
    client,
    task,
  }) => {
    const route = `${baseUrl}/api/v3/openmeter/meters/01METER`
    fetchMock.route(
      route,
      { status: 204 },
      { method: 'DELETE', name: task.name },
    )

    await client.v3.meters.delete('01METER')
    expect(fetchMock.callHistory.done(task.name)).toBeTruthy()
  })

  it<Context>('list billing profiles: GET /openmeter/profiles', async ({
    baseUrl,
    client,
    task,
  }) => {
    const route = `${baseUrl}/api/v3/openmeter/profiles`
    const page = { data: [], meta: { page: 1, page_size: 20, total_count: 0 } }
    fetchMock.route(
      `begin:${route}`,
      { body: page, status: 200 },
      { method: 'GET', name: task.name },
    )

    const resp = await client.v3.billingProfiles.list({
      page: { number: 1, size: 20 },
    })

    expect(resp).toEqual(page)
    expect(fetchMock.callHistory.done(task.name)).toBeTruthy()
  })

  it<Context>('list currencies: GET /openmeter/currencies', async ({
    baseUrl,
    client,
    task,
  }) => {
    const route = `${baseUrl}/api/v3/openmeter/currencies`
    const page = { data: [], meta: { page: 1, page_size: 20, total_count: 0 } }
    fetchMock.route(
      `begin:${route}`,
      { body: page, status: 200 },
      { method: 'GET', name: task.name },
    )

    const resp = await client.v3.currencies.list()

    expect(resp).toEqual(page)
    expect(fetchMock.callHistory.done(task.name)).toBeTruthy()
  })

  it<Context>('create tax code: POST /openmeter/tax-codes with snake_case body', async ({
    baseUrl,
    client,
    task,
  }) => {
    const route = `${baseUrl}/api/v3/openmeter/tax-codes`
    const created = { id: '01J00000000000000000000TAX', key: 'standard' }
    fetchMock.route(
      route,
      { body: created, status: 201 },
      { method: 'POST', name: task.name },
    )

    const resp = await client.v3.taxCodes.create({
      app_mappings: [],
      key: 'standard',
      name: 'Standard',
    })

    expect(resp).toMatchObject({ id: created.id, key: 'standard' })
    const sent = JSON.parse(
      String(fetchMock.callHistory.calls()[0]?.options?.body),
    )
    // snake_case body field passed through verbatim (Option A).
    expect(sent.app_mappings).toEqual([])
    expect(sent.key).toBe('standard')
  })

  it<Context>('list llm cost prices: GET /openmeter/llm-cost/prices', async ({
    baseUrl,
    client,
    task,
  }) => {
    const route = `${baseUrl}/api/v3/openmeter/llm-cost/prices`
    const page = { data: [], meta: { page: 1, page_size: 20, total_count: 0 } }
    fetchMock.route(
      `begin:${route}`,
      { body: page, status: 200 },
      { method: 'GET', name: task.name },
    )

    const resp = await client.v3.llmCost.listPrices()

    expect(resp).toEqual(page)
    expect(fetchMock.callHistory.done(task.name)).toBeTruthy()
  })

  it<Context>('list apps: GET /openmeter/apps', async ({
    baseUrl,
    client,
    task,
  }) => {
    const route = `${baseUrl}/api/v3/openmeter/apps`
    const page = { data: [], meta: { page: 1, page_size: 20, total_count: 0 } }
    fetchMock.route(
      `begin:${route}`,
      { body: page, status: 200 },
      { method: 'GET', name: task.name },
    )

    const resp = await client.v3.apps.list()

    expect(resp).toEqual(page)
    expect(fetchMock.callHistory.done(task.name)).toBeTruthy()
  })

  it<Context>('query governance access: POST /openmeter/governance/query with snake_case body', async ({
    baseUrl,
    client,
    task,
  }) => {
    const route = `${baseUrl}/api/v3/openmeter/governance/query`
    const result = { data: [] }
    fetchMock.route(
      `begin:${route}`,
      { body: result, status: 200 },
      { method: 'POST', name: task.name },
    )

    const resp = await client.v3.governance.queryAccess({
      customer: { keys: ['cust-1'] },
      include_credits: true,
    })

    expect(resp).toEqual(result)
    const sent = JSON.parse(
      String(fetchMock.callHistory.calls()[0]?.options?.body),
    )
    // snake_case body field passed through verbatim (Option A).
    expect(sent.include_credits).toBe(true)
    expect(sent.customer.keys).toEqual(['cust-1'])
  })

  it<Context>('get organization default tax codes: GET /openmeter/defaults/tax-codes', async ({
    baseUrl,
    client,
    task,
  }) => {
    const route = `${baseUrl}/api/v3/openmeter/defaults/tax-codes`
    const data = { vat: 'standard' }
    fetchMock.route(
      route,
      { body: data, status: 200 },
      { method: 'GET', name: task.name },
    )

    const resp = await client.v3.defaults.getOrganizationTaxCodes()

    expect(resp).toEqual(data)
    expect(fetchMock.callHistory.done(task.name)).toBeTruthy()
  })

  it<Context>('list plan addons: GET /openmeter/plans/{planId}/addons', async ({
    baseUrl,
    client,
    task,
  }) => {
    const route = `${baseUrl}/api/v3/openmeter/plans/01PLAN/addons`
    const page = { data: [], meta: { page: 1, page_size: 20, total_count: 0 } }
    fetchMock.route(
      `begin:${route}`,
      { body: page, status: 200 },
      { method: 'GET', name: task.name },
    )

    const resp = await client.v3.plans.listAddons('01PLAN', {
      page: { number: 1, size: 20 },
    })

    expect(resp).toEqual(page)
    expect(fetchMock.callHistory.done(task.name)).toBeTruthy()
  })

  it<Context>('create credit grant: POST /openmeter/customers/{customerId}/credits/grants with snake_case body', async ({
    baseUrl,
    client,
    task,
  }) => {
    const route = `${baseUrl}/api/v3/openmeter/customers/01CUST/credits/grants`
    const created = { id: '01J0000000000000000000GRANT' }
    fetchMock.route(
      route,
      { body: created, status: 201 },
      { method: 'POST', name: task.name },
    )

    const resp = await client.v3.customers.createCreditGrant('01CUST', {
      amount: '100',
      currency: 'USD',
      funding_method: 'none',
      name: 'Welcome credits',
    })

    expect(resp).toMatchObject({ id: created.id })
    const sent = JSON.parse(
      String(fetchMock.callHistory.calls()[0]?.options?.body),
    )
    // snake_case body field passed through verbatim (Option A).
    expect(sent.funding_method).toBe('none')
    expect(sent.amount).toBe('100')
  })

  it<Context>('list subscription addons: GET /openmeter/subscriptions/{subscriptionId}/addons', async ({
    baseUrl,
    client,
    task,
  }) => {
    const route = `${baseUrl}/api/v3/openmeter/subscriptions/01SUB/addons`
    const page = { data: [], meta: { page: 1, page_size: 20, total_count: 0 } }
    fetchMock.route(
      `begin:${route}`,
      { body: page, status: 200 },
      { method: 'GET', name: task.name },
    )

    const resp = await client.v3.subscriptions.listAddons('01SUB', {
      page: { number: 1, size: 20 },
    })

    expect(resp).toEqual(page)
    expect(fetchMock.callHistory.done(task.name)).toBeTruthy()
  })
})
