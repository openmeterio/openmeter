import {
  createLinterRuleTester,
  createTestHost,
  createTestRunner,
} from '@typespec/compiler/testing'
import { beforeEach, describe, it } from 'node:test'
import { casingErrorsRule, casingRule } from '../lib/rules/casing.js'

const enumValueDiagnostic = {
  code: '@openmeter/api-spec-aip/casing-aip-errors',
  message:
    'The values of Enum Value types must use snake_case (dot-separated paths allowed)',
}

describe('casingErrorsRule', () => {
  let ruleTester

  beforeEach(async () => {
    const host = await createTestHost()
    const runner = await createTestRunner(host)
    ruleTester = createLinterRuleTester(
      runner,
      casingErrorsRule,
      '@openmeter/api-spec-aip',
    )
  })

  it('accepts snake_case enum values', async () => {
    await ruleTester
      .expect(
        `enum ChargesExpand {
          RealTimeUsage: "real_time_usage",
          Invoice: "invoice",
        }`,
      )
      .toBeValid()
  })

  it('accepts dot-separated paths of snake_case segments', async () => {
    await ruleTester
      .expect(
        `enum ChargesExpand {
          RealizationTotals: "realization.totals",
          RealizationDetailedLines: "realization.detailed_lines",
        }`,
      )
      .toBeValid()
  })

  it('accepts multi-level paths', async () => {
    await ruleTester
      .expect(`enum E { Threshold: "entitlements.balance.threshold" }`)
      .toBeValid()
  })

  it('rejects non-snake_case path segments', async () => {
    await ruleTester
      .expect(`enum E { RealizationTotals: "Realization.Totals" }`)
      .toEmitDiagnostics(enumValueDiagnostic)
  })

  it('rejects empty path segments', async () => {
    await ruleTester
      .expect(`enum E { RealizationTotals: "realization..totals" }`)
      .toEmitDiagnostics(enumValueDiagnostic)
  })

  it('rejects a leading dot', async () => {
    await ruleTester
      .expect(`enum E { Totals: ".totals" }`)
      .toEmitDiagnostics(enumValueDiagnostic)
  })

  it('rejects a trailing dot', async () => {
    await ruleTester
      .expect(`enum E { Realization: "realization." }`)
      .toEmitDiagnostics(enumValueDiagnostic)
  })

  it('rejects kebab-case enum values', async () => {
    await ruleTester
      .expect(`enum E { RealizationTotals: "realization-totals" }`)
      .toEmitDiagnostics(enumValueDiagnostic)
  })
})

describe('casingRule', () => {
  let ruleTester

  beforeEach(async () => {
    const host = await createTestHost()
    const runner = await createTestRunner(host)
    ruleTester = createLinterRuleTester(
      runner,
      casingRule,
      '@openmeter/api-spec-aip',
    )
  })

  it('accepts snake_case model property names', async () => {
    await ruleTester.expect(`model Charge { invoice_id: string }`).toBeValid()
  })

  // The dot relaxation is scoped to enum wire values; a dot in a property name
  // is still a violation.
  it('rejects dots in model property names', async () => {
    await ruleTester
      .expect('model Charge { `realization.totals`: string }')
      .toEmitDiagnostics({
        code: '@openmeter/api-spec-aip/casing',
        message: 'The names of Model Property types must use snake_case',
      })
  })
})
