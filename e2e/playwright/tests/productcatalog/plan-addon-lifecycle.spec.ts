/**
 * Plan + addon > smoke lifecycle
 *
 * Sequential lifecycle smoke covering the v3 product-catalog plan/addon
 * happy path: provision a meter and feature, create a draft plan with a
 * flat rate card, attach unit and graduated rate cards, reject a
 * duplicate-key rate card, then create + publish an addon, attach it to
 * the plan, and publish the plan.
 *
 * Endpoints exercised:
 *   POST /api/v3/openmeter/meters
 *   POST /api/v3/openmeter/features
 *   POST /api/v3/openmeter/plans (+ PUT, + GET, + /publish)
 *   POST /api/v3/openmeter/addons (+ /publish)
 *   POST /api/v3/openmeter/plans/{planId}/addons
 *
 * Tests run serially (test.describe.configure mode 'serial') and share
 * state via outer-scope variables; an early failure intentionally fails
 * the rest.
 */
import { test, expect } from '@playwright/test'
import { faker } from '@faker-js/faker'
import { BASE, createMeter, createFeature } from '../../helpers/catalog'

test.describe.configure({ mode: 'serial' })

test.describe('Plan + addon > smoke lifecycle', () => {
  let meterId: string
  let featureId: string
  let planId: string
  let addonId: string
  let planAddonId: string
  let planName: string
  let phaseKey: string

  // Rate card keys reused across mutation steps so we can identify them in
  // responses and remove the invalid one in step 5.
  const flatKey = `flat_fee_${faker.string.alphanumeric({ length: 6, casing: 'lower' })}`
  const usageKey = `usage_unit_${faker.string.alphanumeric({ length: 6, casing: 'lower' })}`
  const graduatedKey = `tiered_graduated_${faker.string.alphanumeric({ length: 6, casing: 'lower' })}`

  function flatRateCard(key: string) {
    return {
      key,
      name: 'Monthly Fee',
      price: { type: 'flat', amount: '10' },
      billing_cadence: 'P1M',
      payment_term: 'in_advance',
    }
  }

  function unitRateCard(key: string, featureRefId: string) {
    return {
      key,
      name: 'Usage Unit',
      feature: { id: featureRefId },
      price: { type: 'unit', amount: '0.10' },
      billing_cadence: 'P1M',
      payment_term: 'in_arrears',
    }
  }

  function graduatedRateCard(key: string) {
    return {
      key,
      name: 'Tiered Graduated',
      price: {
        type: 'graduated',
        tiers: [
          { up_to_amount: '100', unit_price: { type: 'unit', amount: '0.10' } },
          { unit_price: { type: 'unit', amount: '0.05' } },
        ],
      },
      billing_cadence: 'P1M',
      payment_term: 'in_arrears',
    }
  }

  // Invalid because the key duplicates an existing rate card in the same phase.
  // Duplicate rate-card keys are rejected at create/update time with HTTP 400.
  function duplicateKeyRateCard(duplicateOf: string) {
    return {
      key: duplicateOf,
      name: 'Duplicate Key Card',
      price: { type: 'flat', amount: '99' },
      billing_cadence: 'P1M',
      payment_term: 'in_advance',
    }
  }

  test('provisions a meter and feature', async ({ request }) => {
    const meter = await createMeter(request)
    expect(meter.id).toBeTruthy()
    meterId = meter.id

    const feature = await createFeature(request, { meterId })
    expect(feature.id).toBeTruthy()
    featureId = feature.id
  })

  test('creates a draft plan with an initial flat rate card', async ({ request }) => {
    expect(meterId).toBeTruthy()

    planName = `Smoke Plan ${faker.string.alphanumeric({ length: 6, casing: 'lower' })}`
    phaseKey = `phase_${faker.string.alphanumeric({ length: 6, casing: 'lower' })}`

    const body = {
      key: `plan_${faker.string.alphanumeric({ length: 12, casing: 'lower' })}`,
      name: planName,
      currency: 'USD',
      billing_cadence: 'P1M',
      phases: [
        {
          key: phaseKey,
          name: 'Phase 1',
          rate_cards: [flatRateCard(flatKey)],
        },
      ],
    }

    const res = await request.post(`${BASE}/plans`, { data: body })
    expect(res.status(), `plan create failed: ${await res.text()}`).toBe(201)
    const plan = await res.json()
    expect(plan.status).toBe('draft')
    expect(plan.phases[0].rate_cards).toHaveLength(1)
    planId = plan.id
  })

  test('adds usage and graduated rate cards via update', async ({ request }) => {
    expect(planId).toBeTruthy()

    const body = {
      name: planName,
      phases: [
        {
          key: phaseKey,
          name: 'Phase 1',
          rate_cards: [
            flatRateCard(flatKey),
            unitRateCard(usageKey, featureId),
            graduatedRateCard(graduatedKey),
          ],
        },
      ],
    }

    const res = await request.put(`${BASE}/plans/${planId}`, { data: body })
    expect(res.status(), `plan update failed: ${await res.text()}`).toBe(200)
    const plan = await res.json()
    expect(plan.phases[0].rate_cards).toHaveLength(3)
    const keys = plan.phases[0].rate_cards.map((rc: { key: string }) => rc.key).sort()
    expect(keys).toEqual([flatKey, graduatedKey, usageKey].sort())
  })

  test('rejects adding an invalid rate card', async ({ request }) => {
    expect(planId).toBeTruthy()

    const body = {
      name: planName,
      phases: [
        {
          key: phaseKey,
          name: 'Phase 1',
          rate_cards: [
            flatRateCard(flatKey),
            unitRateCard(usageKey, featureId),
            graduatedRateCard(graduatedKey),
            duplicateKeyRateCard(flatKey),
          ],
        },
      ],
    }

    const res = await request.put(`${BASE}/plans/${planId}`, { data: body })
    expect(res.status()).toBe(400)
    const problem = await res.json()
    expect(problem.title).toBe('Bad Request')
    expect((problem.detail ?? '').toLowerCase()).toContain('duplicat')
  })

  test('confirms the invalid card was not added', async ({ request }) => {
    expect(planId).toBeTruthy()

    const res = await request.get(`${BASE}/plans/${planId}`)
    expect(res.status(), `plan get failed: ${await res.text()}`).toBe(200)
    const plan = await res.json()
    expect(plan.phases[0].rate_cards).toHaveLength(3)
    const keys = plan.phases[0].rate_cards.map((rc: { key: string }) => rc.key).sort()
    expect(keys).toEqual([flatKey, graduatedKey, usageKey].sort())
  })

  test('creates a draft addon', async ({ request }) => {
    const body = {
      key: `addon_${faker.string.alphanumeric({ length: 12, casing: 'lower' })}`,
      name: `Smoke Addon ${faker.string.alphanumeric({ length: 6, casing: 'lower' })}`,
      currency: 'USD',
      instance_type: 'single',
      rate_cards: [flatRateCard(`addon_fee_${faker.string.alphanumeric({ length: 6, casing: 'lower' })}`)],
    }

    const res = await request.post(`${BASE}/addons`, { data: body })
    expect(res.status(), `addon create failed: ${await res.text()}`).toBe(201)
    const addon = await res.json()
    expect(addon.status).toBe('draft')
    addonId = addon.id
  })

  test('publishes the addon', async ({ request }) => {
    expect(addonId).toBeTruthy()
    const res = await request.post(`${BASE}/addons/${addonId}/publish`)
    expect(res.status(), `addon publish failed: ${await res.text()}`).toBe(200)
    const addon = await res.json()
    expect(addon.status).toBe('active')
    expect(addon.effective_from).toBeTruthy()
  })

  test('attaches the addon to the plan', async ({ request }) => {
    expect(planId).toBeTruthy()
    expect(addonId).toBeTruthy()

    const body = {
      name: 'Smoke Plan Addon',
      addon: { id: addonId },
      from_plan_phase: phaseKey,
    }

    const res = await request.post(`${BASE}/plans/${planId}/addons`, { data: body })
    expect(res.status(), `plan-addon attach failed: ${await res.text()}`).toBe(201)
    const planAddon = await res.json()
    expect(planAddon.addon.id).toBe(addonId)
    expect(planAddon.from_plan_phase).toBe(phaseKey)
    planAddonId = planAddon.id
    expect(planAddonId).toBeTruthy()
  })

  test('publishes the plan', async ({ request }) => {
    expect(planId).toBeTruthy()
    const res = await request.post(`${BASE}/plans/${planId}/publish`)
    expect(res.status(), `plan publish failed: ${await res.text()}`).toBe(200)
    const plan = await res.json()
    expect(plan.status).toBe('active')
    expect(plan.effective_from).toBeTruthy()
  })
})
