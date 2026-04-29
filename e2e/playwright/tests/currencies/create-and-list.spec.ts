import { test, expect } from '@playwright/test'
import { faker } from '@faker-js/faker'

const BASE = '/api/v3/openmeter'

// Currency codes must be 3-24 chars and not conflict with fiat codes (USD, EUR, …)
function uniqueCurrencyCode(): string {
  return `TST${faker.string.alphanumeric({ length: 5, casing: 'upper' })}`
}

test.describe('Currencies > create and list', () => {
  test('creates a custom currency and finds it in the list', async ({ request }) => {
    const code = uniqueCurrencyCode()
    const name = faker.string.uuid()

    const createRes = await request.post(`${BASE}/currencies/custom`, {
      // symbol is required server-side despite being optional in the OpenAPI spec
      data: { code, name, description: 'A custom currency for testing', symbol: '¤' },
    })
    expect(createRes.status()).toBe(201)
    const currency = await createRes.json()
    expect(currency.id).toBeTruthy()
    expect(currency.type).toBe('custom')
    expect(currency.code).toBe(code)
    expect(currency.name).toBe(name)
    expect(currency.created_at).toBeTruthy()

    const listRes = await request.get(`${BASE}/currencies`, {
      params: { 'page[size]': '1000' },
    })
    expect(listRes.status()).toBe(200)
    const { data } = await listRes.json()
    const found = data.find((c: { id: string }) => c.id === currency.id)
    expect(found).toBeDefined()
    expect(found.type).toBe('custom')
    expect(found.code).toBe(code)
  })

  test('lists only custom currencies when filter[type]=custom', async ({ request }) => {
    const createRes = await request.post(`${BASE}/currencies/custom`, {
      // symbol is required server-side despite being optional in the OpenAPI spec
      data: { code: uniqueCurrencyCode(), name: faker.string.uuid(), symbol: '¤' },
    })
    expect(createRes.status()).toBe(201)
    const currency = await createRes.json()

    const listRes = await request.get(`${BASE}/currencies`, {
      params: { 'page[size]': '1000', 'filter[type]': 'custom' },
    })
    expect(listRes.status()).toBe(200)
    const { data } = await listRes.json()
    for (const item of data) {
      expect(item.type).toBe('custom')
    }
    const found = data.find((c: { id: string }) => c.id === currency.id)
    expect(found).toBeDefined()
  })

  test('lists only fiat currencies when filter[type]=fiat', async ({ request }) => {
    const listRes = await request.get(`${BASE}/currencies`, {
      params: { 'page[size]': '1000', 'filter[type]': 'fiat' },
    })
    expect(listRes.status()).toBe(200)
    const { data } = await listRes.json()
    expect(data.length).toBeGreaterThan(0)
    for (const item of data) {
      expect(item.type).toBe('fiat')
    }
  })

  for (const { label, buildBody } of [
    { label: 'missing required name', buildBody: () => ({ code: uniqueCurrencyCode(), symbol: '¤' }) },
    { label: 'missing required code', buildBody: () => ({ name: faker.string.uuid(), symbol: '¤' }) },
  ]) {
    test(`rejects create with ${label}`, async ({ request }) => {
      const res = await request.post(`${BASE}/currencies/custom`, { data: buildBody() })
      expect(res.status()).toBe(400)
      const problem = await res.json()
      const rules = (problem.invalid_parameters ?? []).map((p: any) => p.rule)
      expect(rules).toContain('required')
    })
  }
})
