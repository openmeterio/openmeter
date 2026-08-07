import { describe, expect, it } from 'vitest'
import { fromWire, toWire } from '../src/lib/wire.js'
import {
  chargeRealization,
  chargeRealizationDetailedLineUsageBased,
  invoiceReference,
  invoiceStandard,
} from '../src/models/schemas.js'
import { collectFieldKeys, sampleCamel, sampleSnake } from './wire-helpers.js'

// The charge realization is where the expandable-reference shapes the spend API
// introduced actually land: `invoice` is `Invoice | InvoiceReference`, and
// `Invoice` is itself the discriminated union of its concrete types — a union
// nested inside a union. `detailed_lines` (the `realization.detailed_lines`
// expand) is an array of another discriminated union.
//
// The per-operation sweep in wire.generated.spec.ts cannot pin this: its samples
// take a union's FIRST option, so the reference side of every expandable
// reference is never exercised, and nothing there asserts that the expanded side
// keeps its fields rather than collapsing onto the reference.

const asRecord = (value: unknown): Record<string, unknown> =>
  value as Record<string, unknown>

/** A wire-shaped realization with `invoice` forced to the given wire value. */
function realizationWith(invoice: unknown): Record<string, unknown> {
  return { ...asRecord(sampleSnake(chargeRealization)), invoice }
}

describe('charge realization wire mapping', () => {
  it('keeps the expanded invoice instead of collapsing it onto the reference', () => {
    const expanded = asRecord(sampleSnake(invoiceStandard))
    const wire = realizationWith(expanded)

    const mapped = asRecord(fromWire(wire, chargeRealization))
    const invoice = asRecord(mapped.invoice)

    // The nested union has to resolve before it can compete with the reference
    // variant. If it does not, the reference wins on key coverage and every
    // field of the expanded invoice but `id` is dropped.
    expect(Object.keys(invoice).length).toBeGreaterThan(1)
    expect(invoice.type).toBe(asRecord(expanded).type)
    expect(invoice.id).toBe(asRecord(expanded).id)
    expect(toWire(mapped, chargeRealization)).toEqual(wire)
  })

  it('maps the invoice reference, the variant the operation sweep never samples', () => {
    const wire = realizationWith(sampleSnake(invoiceReference))

    const mapped = asRecord(fromWire(wire, chargeRealization))
    expect(Object.keys(asRecord(mapped.invoice))).toEqual(['id'])
    expect(toWire(mapped, chargeRealization)).toEqual(wire)
  })

  it('maps the detailed lines the realization.detailed_lines expand adds', () => {
    const line = asRecord(sampleCamel(chargeRealizationDetailedLineUsageBased))
    const publicValue = {
      ...asRecord(sampleCamel(chargeRealization)),
      detailedLines: [line],
    }

    const wire = asRecord(toWire(publicValue, chargeRealization))
    const wireLines = wire.detailed_lines as Record<string, unknown>[]

    expect(wireLines).toHaveLength(1)
    // A detailed line carries multi-word keys in both cases; the walk has to
    // descend through the array AND the line's own discriminated union to rename
    // them, so a leaked camelCase key here means one of those steps was skipped.
    expect(collectFieldKeys(wireLines).filter((k) => /[A-Z]/.test(k))).toEqual(
      [],
    )
    expect(fromWire(wire, chargeRealization)).toEqual(publicValue)
  })

  it('leaks no wire casing across a fully expanded realization', () => {
    const wire = realizationWith(sampleSnake(invoiceStandard))
    const mapped = fromWire(wire, chargeRealization)

    expect(collectFieldKeys(mapped).filter((k) => /_/.test(k))).toEqual([])
    expect(toWire(mapped, chargeRealization)).toEqual(wire)
  })
})
