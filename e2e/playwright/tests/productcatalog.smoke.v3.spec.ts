import { expect, test } from '@playwright/test'
import { isHTTPError, OpenMeter } from '@openmeter/sdk'

// Live-server smoke for the @openmeter/sdk v3 compatibility shim (`om.v3.*`).
//
// This is the TypeScript parity of e2e/productcatalog_smoke_v3_test.go: it
// exercises the same cross-cutting plan + addon authoring flow end to end, but
// through the shim instead of raw Go HTTP. Where the Go test asserts the v3
// wire shape directly, this one asserts the same shape coming back through the
// shim — which surfaces the wire verbatim (snake_case, "Option A"), so the
// fields read identically (status, phases[].rate_cards, validation_errors, …).
//
// Flow (identical to the Go smoke):
//   - Create two meters + two features bound to them.
//   - Create a draft plan with a single flat rate card.
//   - Update the plan to carry flat + usage (with feature) + graduated rate cards.
//   - Add a defective rate card (cadence-misaligned) and confirm the
//     draft-with-errors loop: GET surfaces validation_errors, publish is
//     rejected with the same code.
//   - Remove the defective rate card and confirm validation_errors clears.
//   - Create a draft addon and publish it.
//   - Attach the published addon to the still-draft plan.
//   - Publish the plan and confirm the attached addon survives the transition.
//
// Requires a running OpenMeter (no auth locally). Point it at the server with
// OPENMETER_ADDRESS (defaults to the docker-compose port used by `make -C e2e`).

const address = process.env.OPENMETER_ADDRESS ?? 'http://localhost:38888'

// The shim appends `/api/v3` and resolves the Bearer header only when apiKey is
// set, so the bare local server (no auth) needs only baseUrl.
const om = new OpenMeter({ baseUrl: address })

// uniqueKey mirrors the Go helper: a fixture key that survives re-runs against a
// shared database without collision.
function uniqueKey(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 10_000)}`
}

// Derive the exact request types from the shim's method signatures — the v3
// schema types aren't re-exported from the package root, so we pull them off the
// methods instead. RateCard is the element type shared by plan rate-card arrays;
// the explicit return type narrows the discriminated `price`/`payment_term`
// literals without `as const`.
type CreatePlanBody = Parameters<typeof om.v3.plans.create>[0]
type RateCard = CreatePlanBody['phases'][number]['rate_cards'][number]

// --- Fixture builders (parity with e2e/v3helpers_test.go) ---

function validFlatRateCard(keyPrefix: string): RateCard {
  return {
    key: uniqueKey(keyPrefix),
    name: `Test Rate Card ${keyPrefix}`,
    price: { type: 'flat', amount: '10' },
    billing_cadence: 'P1M',
    payment_term: 'in_advance',
  }
}

function validUnitRateCard(feature: { id: string; key: string }): RateCard {
  return {
    key: feature.key,
    name: `Test Unit Rate Card ${feature.key}`,
    price: { type: 'unit', amount: '0.10' },
    billing_cadence: 'P1M',
    payment_term: 'in_arrears',
    feature: { id: feature.id },
  }
}

function validGraduatedRateCard(feature: { id: string; key: string }): RateCard {
  return {
    key: feature.key,
    name: `Test Graduated Rate Card ${feature.key}`,
    price: {
      type: 'graduated',
      tiers: [
        { up_to_amount: '100', unit_price: { type: 'unit', amount: '0.10' } },
        { unit_price: { type: 'unit', amount: '0.05' } },
      ],
    },
    billing_cadence: 'P1M',
    payment_term: 'in_arrears',
    feature: { id: feature.id },
  }
}

// validationCodes pulls the product-catalog validation codes out of a thrown
// HTTPError's problem extensions — the shim analogue of the Go
// v3Problem.ValidationErrors() helper.
function validationCodes(err: unknown): string[] {
  if (!isHTTPError(err)) return []
  const extensions = err.getField('extensions') as
    | { validationErrors?: Array<{ code?: string }> }
    | undefined
  return (extensions?.validationErrors ?? [])
    .map((e) => e.code)
    .filter((c): c is string => typeof c === 'string')
}

test.describe.serial('v3 shim product catalog smoke', () => {
  const meters: Array<{ id: string }> = []
  const features: Array<{ id: string; key: string }> = []

  let planId = ''
  let addonId = ''
  let planAddonId = ''
  let phaseKey = ''
  // Track the three valid rate cards across the invalid-loop steps so the
  // "remove defective" PUT can rebuild the phase from the same baseline.
  let validRateCards: RateCard[] = []

  test.beforeAll(async () => {
    // given:
    // - two meters and two features bound to them, the raw materials the plan's
    //   usage-based rate cards reference.
    for (let i = 0; i < 2; i++) {
      const meterKey = uniqueKey('sanity_meter')
      const meter = (await om.v3.meters.create({
        key: meterKey,
        name: `Test Meter ${meterKey}`,
        aggregation: 'sum',
        event_type: uniqueKey('sanity_event'),
        value_property: '$.value',
      })) as { id: string }
      expect(meter.id).toBeTruthy()
      meters.push(meter)

      const featureKey = uniqueKey('sanity_feature')
      const feature = (await om.v3.features.create({
        key: featureKey,
        name: `Test Feature ${featureKey}`,
        meter: { id: meter.id },
      })) as { id: string; key: string }
      expect(feature.id).toBeTruthy()
      features.push(feature)
    }
  })

  test('creates a draft plan with a single flat rate card', async () => {
    // when:
    // - a plan is created with the baseline single-flat-rate-card phase.
    phaseKey = uniqueKey('phase_1')
    const plan = (await om.v3.plans.create({
      key: uniqueKey('sanity_plan'),
      name: 'Sanity Plan',
      currency: 'USD',
      billing_cadence: 'P1M',
      phases: [
        {
          key: phaseKey,
          name: 'Sanity Phase',
          rate_cards: [validFlatRateCard('fee')],
        },
      ],
    })) as { id: string; status: string; phases: Array<{ rate_cards: unknown[] }> }

    // then:
    // - it lands as a draft with the single rate card.
    expect(plan.id).toBeTruthy()
    expect(plan.status).toBe('draft')
    expect(plan.phases).toHaveLength(1)
    expect(plan.phases[0].rate_cards).toHaveLength(1)
    planId = plan.id
  })

  test('updates the plan to carry flat + usage + graduated rate cards', async () => {
    // given:
    // - three different rate card shapes on one phase. The flat fee is
    //   in_advance; both usage-based ones are in_arrears (unit/graduated prices
    //   cannot be in_advance). Only the unit one carries a feature reference.
    const flat = validFlatRateCard('sanity_flat')
    const usage = validUnitRateCard(features[0])
    const graduated = validGraduatedRateCard(features[1])

    // when:
    const plan = (await om.v3.plans.update(planId, {
      name: 'Sanity Plan',
      phases: [
        {
          key: phaseKey,
          name: 'Sanity Phase',
          rate_cards: [flat, usage, graduated],
        },
      ],
    })) as {
      phases: Array<{
        rate_cards: Array<{ key: string; feature?: { id: string } }>
      }>
    }

    // then:
    // - all three rate cards round-trip and the usage card keeps its feature.
    expect(plan.phases).toHaveLength(1)
    expect(plan.phases[0].rate_cards).toHaveLength(3)

    const usageRC = plan.phases[0].rate_cards.find((rc) => rc.key === usage.key)
    expect(usageRC, 'usage rate card missing after update').toBeTruthy()
    expect(
      usageRC?.feature?.id,
      'usage rate card lost its feature binding after update',
    ).toBe(usage.feature?.id)
  })

  test('adds a defective rate card and surfaces validation_errors', async () => {
    // given:
    // - the current valid rate cards read back from the plan, so we don't drift
    //   from server-normalized values (e.g. "0.10" -> "0.1").
    const current = (await om.v3.plans.get(planId)) as {
      phases: Array<{ rate_cards: RateCard[] }>
    }
    expect(current.phases[0].rate_cards).toHaveLength(3)
    validRateCards = current.phases[0].rate_cards

    // - a defective flat rate card whose billing cadence (P2W) doesn't align
    //   with the plan's P1M cadence — picked because it surfaces a single
    //   actionable validation error with a useful field path.
    const defective: RateCard = {
      ...validFlatRateCard('defective_cadence'),
      billing_cadence: 'P2W',
    }

    // when:
    // - updating the draft with the defective card is accepted (drafts may carry
    //   validation errors).
    await om.v3.plans.update(planId, {
      name: 'Sanity Plan',
      phases: [
        {
          key: phaseKey,
          name: 'Sanity Phase',
          rate_cards: [...validRateCards, defective],
        },
      ],
    })

    // then:
    // - GET surfaces the cadence-unaligned validation error on the draft.
    const got = (await om.v3.plans.get(planId)) as {
      validation_errors?: Array<{ code: string }>
    }
    expect(got.validation_errors, 'expected validation_errors on the draft').toBeTruthy()
    const codes = (got.validation_errors ?? []).map((e) => e.code)
    expect(codes).toContain('rate_card_billing_cadence_unaligned')

    // - publish is rejected with the same code (400).
    let publishErr: unknown
    try {
      await om.v3.plans.publish(planId)
    } catch (e) {
      publishErr = e
    }
    expect(isHTTPError(publishErr), 'publish should reject the defective draft').toBe(true)
    expect((publishErr as { status: number }).status).toBe(400)
    expect(validationCodes(publishErr)).toContain('rate_card_billing_cadence_unaligned')
  })

  test('removes the defective rate card and clears validation_errors', async () => {
    // when:
    // - the phase is rebuilt from the known-good baseline.
    const plan = (await om.v3.plans.update(planId, {
      name: 'Sanity Plan',
      phases: [
        {
          key: phaseKey,
          name: 'Sanity Phase',
          rate_cards: validRateCards,
        },
      ],
    })) as { phases: Array<{ rate_cards: unknown[] }> }

    // then:
    // - the three valid cards remain and validation_errors clears.
    expect(plan.phases[0].rate_cards).toHaveLength(3)

    const got = (await om.v3.plans.get(planId)) as {
      validation_errors?: unknown[]
    }
    if (got.validation_errors != null) {
      expect(
        got.validation_errors,
        'expected validation_errors to clear after removing the defective rate card',
      ).toHaveLength(0)
    }
  })

  test('creates a draft addon', async () => {
    // when:
    const addon = (await om.v3.addons.create({
      key: uniqueKey('sanity_addon'),
      name: 'Test Addon sanity_addon',
      currency: 'USD',
      instance_type: 'single',
      rate_cards: [validFlatRateCard('addon_fee')],
    })) as { id: string; status: string }

    // then:
    expect(addon.id).toBeTruthy()
    expect(addon.status).toBe('draft')
    addonId = addon.id
  })

  test('publishes the addon', async () => {
    const addon = (await om.v3.addons.publish(addonId)) as { status: string }
    expect(addon.status).toBe('active')
  })

  test('attaches the published addon to the plan', async () => {
    // when:
    const planAddon = (await om.v3.plans.createAddon(planId, {
      name: 'Test Plan Addon',
      addon: { id: addonId },
      from_plan_phase: phaseKey,
    })) as { id: string; addon: { id: string }; from_plan_phase: string }

    // then:
    expect(planAddon.addon.id).toBe(addonId)
    expect(planAddon.from_plan_phase).toBe(phaseKey)
    planAddonId = planAddon.id
  })

  test('publishes the plan and keeps the attached addon', async () => {
    // when:
    const plan = (await om.v3.plans.publish(planId)) as {
      status: string
      effective_from?: unknown
    }

    // then:
    // - the plan goes active with an effective_from window...
    expect(plan.status).toBe('active')
    expect(plan.effective_from).toBeTruthy()

    // - ...and the attached addon survives the transition.
    const page = (await om.v3.plans.listAddons(planId)) as {
      data: Array<{ id: string }>
    }
    const found = page.data.some((pa) => pa.id === planAddonId)
    expect(found, 'attached plan-addon missing after plan publish').toBe(true)
  })
})
