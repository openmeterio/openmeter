import { expect, test } from '@playwright/test'
import {
  type AddonCreate,
  type Feature,
  type FeatureCreateInputs,
  isHTTPError,
  type Meter,
  type MeterCreate,
  OpenMeter,
  type Plan,
  type PlanAddon,
  type PlanAddonCreate,
  type PlanCreate,
  type PlanReplaceUpdate,
  type RateCard,
} from '@openmeter/sdk'

// Live-server smoke for the legacy v1 OpenMeter client (`om.*`).
//
// This is the v1 counterpart of productcatalog.smoke.v3.spec.ts — the SAME
// flow, written against the v1 client so the two can be read side by side. The
// instructive difference is *how the types are obtained and used*:
//
//   v1 (this file): the request/response types are re-exported from the package
//                   root (`export * from './schemas.js'`), so we import them by
//                   name — `PlanCreate`, `RateCard`, `Plan`, `PlanAddon`, … — and
//                   both bodies and responses are fully typed with no casts.
//
//   v3 (.v3 file):  the v3 schema types are NOT re-exported from the root, so the
//                   test has to *derive* request types off the method signatures
//                   (`Parameters<typeof om.v3.plans.create>[0]`) and cast
//                   responses with `as { ... }`.
//
// The wire shapes differ too, which is the other half of the comparison:
//   - v1 is camelCase (`billingCadence`, `rateCards`, `validationErrors`); v3 is
//     snake_case verbatim.
//   - v1 references features by `featureKey` and meters by `meterSlug`/`slug`;
//     v3 references by `feature.id` / `meter.id`.
//   - v1 rate cards are `flat_fee` / `usage_based` wrappers with the payment term
//     nested in the flat price; v3 has a flat price union with `payment_term` on
//     the rate card.
//   - v1 tiered prices use `type: 'tiered'` + `mode: 'graduated'`; v3 uses
//     `type: 'graduated'` directly.
//   - v1 list responses paginate under `items`; v3 under `data`.
//   - v1 `PlanAddon` has no own id — the association is keyed by `addon` +
//     `fromPlanPhase`; v3 has a distinct plan-addon `id`.
//
// Requires a running OpenMeter (no auth locally). Point it at the server with
// OPENMETER_ADDRESS (defaults to the docker-compose port used by `make -C e2e`).

const address = process.env.OPENMETER_ADDRESS ?? 'http://localhost:38888'

// The v1 client targets the bare host (it owns the `/api/v1` path prefix) and
// resolves the Bearer header only when apiKey is set, so the local server (no
// auth) needs only baseUrl.
const om = new OpenMeter({ baseUrl: address })

// uniqueKey mirrors the Go helper: a fixture key that survives re-runs against a
// shared database without collision.
function uniqueKey(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 10_000)}`
}

// defined unwraps the `T | undefined` that the client methods return on success
// (transformResponse returns the parsed body or undefined). Asserting + casting
// here keeps each call site typed as the concrete response model.
function defined<T>(value: T): NonNullable<T> {
  expect(value, 'expected a response body').toBeDefined()
  return value as NonNullable<T>
}

// --- Fixture builders (parity with e2e/v3helpers_test.go, v1 types) ---

function validFlatRateCard(keyPrefix: string): RateCard {
  return {
    type: 'flat_fee',
    key: uniqueKey(keyPrefix),
    name: `Test Rate Card ${keyPrefix}`,
    billingCadence: 'P1M',
    // v1 puts the payment term inside the price, not on the rate card.
    price: { type: 'flat', amount: '10', paymentTerm: 'in_advance' },
  }
}

function validUnitRateCard(feature: Feature): RateCard {
  return {
    type: 'usage_based',
    // v1 references the feature by key (string), not by id.
    key: feature.key,
    name: `Test Unit Rate Card ${feature.key}`,
    featureKey: feature.key,
    billingCadence: 'P1M',
    price: { type: 'unit', amount: '0.10' },
  }
}

function validGraduatedRateCard(feature: Feature): RateCard {
  return {
    type: 'usage_based',
    key: feature.key,
    name: `Test Graduated Rate Card ${feature.key}`,
    featureKey: feature.key,
    billingCadence: 'P1M',
    price: {
      // v1 tiered price: type 'tiered' + mode 'graduated' (v3 uses 'graduated').
      type: 'tiered',
      mode: 'graduated',
      tiers: [
        {
          upToAmount: '100',
          flatPrice: null,
          unitPrice: { type: 'unit', amount: '0.10' },
        },
        { flatPrice: null, unitPrice: { type: 'unit', amount: '0.05' } },
      ],
    },
  }
}

// validationCodes pulls the product-catalog validation codes out of a thrown
// HTTPError's problem extensions — identical helper to the v3 test (both v1 and
// v3 product-catalog validation surface through the same commonhttp envelope:
// extensions.validationErrors).
function validationCodes(err: unknown): string[] {
  if (!isHTTPError(err)) return []
  const extensions = err.getField('extensions') as
    | { validationErrors?: Array<{ code?: string }> }
    | undefined
  return (extensions?.validationErrors ?? [])
    .map((e) => e.code)
    .filter((c): c is string => typeof c === 'string')
}

test.describe.serial('v1 client product catalog smoke', () => {
  const meters: Meter[] = []
  const features: Feature[] = []

  let planId = ''
  let addonId = ''
  let phaseKey = ''
  // Track the three valid rate cards across the invalid-loop steps so the
  // "remove defective" PUT can rebuild the phase from the same baseline.
  let validRateCards: RateCard[] = []

  test.beforeAll(async () => {
    // given:
    // - two meters and two features bound to them (by slug), the raw materials
    //   the plan's usage-based rate cards reference (by featureKey).
    for (let i = 0; i < 2; i++) {
      const slug = uniqueKey('sanity_meter')
      const meterBody: MeterCreate = {
        slug,
        name: `Test Meter ${slug}`,
        aggregation: 'SUM', // v1 aggregation is upper-case (v3 is lower-case).
        eventType: uniqueKey('sanity_event'),
        valueProperty: '$.value',
      }
      const meter = defined(await om.meters.create(meterBody))
      expect(meter.id).toBeTruthy()
      meters.push(meter)

      const featureKey = uniqueKey('sanity_feature')
      const featureBody: FeatureCreateInputs = {
        key: featureKey,
        name: `Test Feature ${featureKey}`,
        meterSlug: slug,
      }
      const feature = defined(await om.features.create(featureBody))
      expect(feature.id).toBeTruthy()
      features.push(feature)
    }
  })

  test('creates a draft plan with a single flat rate card', async () => {
    // when:
    // - a plan is created with the baseline single-flat-rate-card phase.
    phaseKey = uniqueKey('phase_1')
    const body: PlanCreate = {
      key: uniqueKey('sanity_plan'),
      name: 'Sanity Plan',
      currency: 'USD',
      billingCadence: 'P1M',
      phases: [
        {
          key: phaseKey,
          name: 'Sanity Phase',
          duration: null, // last (only) phase runs indefinitely.
          rateCards: [validFlatRateCard('fee')],
        },
      ],
    }
    const plan: Plan = defined(await om.plans.create(body))

    // then:
    // - it lands as a draft with the single rate card.
    expect(plan.id).toBeTruthy()
    expect(plan.status).toBe('draft')
    expect(plan.phases).toHaveLength(1)
    expect(plan.phases[0].rateCards).toHaveLength(1)
    planId = plan.id
  })

  test('updates the plan to carry flat + usage + graduated rate cards', async () => {
    // given:
    // - three different rate card shapes on one phase. The flat fee is
    //   in_advance; both usage-based ones default to in_arrears. Only the unit
    //   one carries a feature reference here.
    const flat = validFlatRateCard('sanity_flat')
    const usage = validUnitRateCard(features[0])
    const graduated = validGraduatedRateCard(features[1])

    // when:
    // - v1 update (PlanReplaceUpdate) requires billingCadence; v3's
    //   UpsertPlanRequest only needs name + phases.
    const body: PlanReplaceUpdate = {
      name: 'Sanity Plan',
      billingCadence: 'P1M',
      phases: [
        {
          key: phaseKey,
          name: 'Sanity Phase',
          duration: null,
          rateCards: [flat, usage, graduated],
        },
      ],
    }
    const plan: Plan = defined(await om.plans.update(planId, body))

    // then:
    // - all three rate cards round-trip and the usage card keeps its feature.
    expect(plan.phases).toHaveLength(1)
    expect(plan.phases[0].rateCards).toHaveLength(3)

    const usageRC = plan.phases[0].rateCards.find((rc) => rc.key === usage.key)
    expect(usageRC, 'usage rate card missing after update').toBeTruthy()
    expect(
      usageRC?.featureKey,
      'usage rate card lost its feature binding after update',
    ).toBe(features[0].key)
  })

  test('adds a defective rate card and surfaces validationErrors', async () => {
    // given:
    // - the current valid rate cards read back from the plan, so we don't drift
    //   from server-normalized values (e.g. "0.10" -> "0.1").
    const current: Plan = defined(await om.plans.get(planId))
    expect(current.phases[0].rateCards).toHaveLength(3)
    validRateCards = current.phases[0].rateCards

    // - a defective flat rate card whose billing cadence (P2W) doesn't align
    //   with the plan's P1M cadence — surfaces a single actionable error.
    const defective: RateCard = {
      ...validFlatRateCard('defective_cadence'),
      billingCadence: 'P2W',
    }

    // when:
    // - updating the draft with the defective card is accepted (drafts may carry
    //   validation errors).
    await om.plans.update(planId, {
      name: 'Sanity Plan',
      billingCadence: 'P1M',
      phases: [
        {
          key: phaseKey,
          name: 'Sanity Phase',
          duration: null,
          rateCards: [...validRateCards, defective],
        },
      ],
    })

    // then:
    // - GET surfaces the cadence-unaligned validation error on the draft.
    const got: Plan = defined(await om.plans.get(planId))
    expect(got.validationErrors, 'expected validationErrors on the draft').toBeTruthy()
    const codes = (got.validationErrors ?? []).map((e) => e.code)
    expect(codes).toContain('rate_card_billing_cadence_unaligned')

    // - publish is rejected with the same code (400).
    let publishErr: unknown
    try {
      await om.plans.publish(planId)
    } catch (e) {
      publishErr = e
    }
    expect(isHTTPError(publishErr), 'publish should reject the defective draft').toBe(true)
    expect((publishErr as { status: number }).status).toBe(400)
    expect(validationCodes(publishErr)).toContain('rate_card_billing_cadence_unaligned')
  })

  test('removes the defective rate card and clears validationErrors', async () => {
    // when:
    // - the phase is rebuilt from the known-good baseline.
    const plan: Plan = defined(
      await om.plans.update(planId, {
        name: 'Sanity Plan',
        billingCadence: 'P1M',
        phases: [
          {
            key: phaseKey,
            name: 'Sanity Phase',
            duration: null,
            rateCards: validRateCards,
          },
        ],
      }),
    )

    // then:
    // - the three valid cards remain and validationErrors clears.
    expect(plan.phases[0].rateCards).toHaveLength(3)

    const got: Plan = defined(await om.plans.get(planId))
    if (got.validationErrors != null) {
      expect(
        got.validationErrors,
        'expected validationErrors to clear after removing the defective rate card',
      ).toHaveLength(0)
    }
  })

  test('creates a draft addon', async () => {
    // when:
    const body: AddonCreate = {
      key: uniqueKey('sanity_addon'),
      name: 'Test Addon sanity_addon',
      currency: 'USD',
      instanceType: 'single',
      rateCards: [validFlatRateCard('addon_fee')],
    }
    const addon = defined(await om.addons.create(body))

    // then:
    expect(addon.id).toBeTruthy()
    expect(addon.status).toBe('draft')
    addonId = addon.id
  })

  test('publishes the addon', async () => {
    const addon = defined(await om.addons.publish(addonId))
    expect(addon.status).toBe('active')
  })

  test('attaches the published addon to the plan', async () => {
    // when:
    const body: PlanAddonCreate = {
      // v1 attaches by addonId (string); v3 nests addon: { id }.
      addonId,
      fromPlanPhase: phaseKey,
    }
    const planAddon: PlanAddon = defined(await om.plans.addons.create(planId, body))

    // then:
    // - v1 PlanAddon has no own id; it carries the full addon + phase key.
    expect(planAddon.addon.id).toBe(addonId)
    expect(planAddon.fromPlanPhase).toBe(phaseKey)
  })

  test('publishes the plan and keeps the attached addon', async () => {
    // when:
    const plan: Plan = defined(await om.plans.publish(planId))

    // then:
    // - the plan goes active with an effectiveFrom window...
    expect(plan.status).toBe('active')
    expect(plan.effectiveFrom).toBeTruthy()

    // - ...and the attached addon survives the transition. v1 paginates under
    //   `items` and the association is keyed by the addon id (no plan-addon id).
    const page = defined(await om.plans.addons.list(planId))
    const found = page.items.some((pa) => pa.addon.id === addonId)
    expect(found, 'attached plan-addon missing after plan publish').toBe(true)
  })
})
