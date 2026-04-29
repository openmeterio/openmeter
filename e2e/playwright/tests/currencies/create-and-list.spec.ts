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

    // Create
    const createRes = await request.post(`${BASE}/currencies/custom`, {
      data: { code, name, description: 'A custom currency for testing', symbol: '¤' },
    })
    expect(createRes.status()).toBe(201)
    const currency = await createRes.json()
    expect(currency.type).toBe('custom')
    expect(currency.code).toBe(code)
    expect(currency.name).toBe(name)
    expect(currency.id).toBeTruthy()

    // List all currencies and find the created one
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
    const code = uniqueCurrencyCode()
    const name = faker.string.uuid()

    // Create a custom currency to ensure at least one exists
    const createRes = await request.post(`${BASE}/currencies/custom`, {
      data: { code, name, symbol: '¤' },
    })
    expect(createRes.status()).toBe(201)
    const currency = await createRes.json()

    // List filtered to custom only
    const listRes = await request.get(`${BASE}/currencies`, {
      params: { 'page[size]': '1000', 'filter[type]': 'custom' },
    })
    expect(listRes.status()).toBe(200)
    const { data } = await listRes.json()

    // Every item must be a custom currency
    for (const item of data) {
      expect(item.type).toBe('custom')
    }

    // The one we created must appear
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

  test('rejects create with missing required name', async ({ request }) => {
    const res = await request.post(`${BASE}/currencies/custom`, {
      data: { code: uniqueCurrencyCode(), symbol: '¤' },
    })
    expect(res.status()).toBe(400)
    const problem = await res.json()
    expect(problem.type).toBeDefined()
  })

  test('rejects create with missing required code', async ({ request }) => {
    const res = await request.post(`${BASE}/currencies/custom`, {
      data: { name: faker.string.uuid(), symbol: '¤' },
    })
    expect(res.status()).toBe(400)
    const problem = await res.json()
    expect(problem.type).toBeDefined()
  })
})
