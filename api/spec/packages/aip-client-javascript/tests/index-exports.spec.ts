import { describe, expect, it } from 'vitest'
import type { Price, UpdateMeterRequest } from '../src/index.js'

// The package root must resolve operation-alias names to the operation envelope,
// not to a raw domain model that shares the name. 18 names used to collide, and
// the explicit model re-export in index.ts silently shadowed the operation alias
// (ESM named exports win over `export type *`) with an incompatible shape —
// e.g. `UpdateMeterRequest` resolved to the bare update body, missing `meterId`.
// Checked at compile time by the tests typecheck; the runtime test only anchors
// the file in the suite.
type _UpdateMeterRequestIsEnvelope = UpdateMeterRequest extends {
  meterId: string
  body: unknown
}
  ? true
  : { __error: 'UpdateMeterRequest lost its operation envelope shape' }
const _updateMeterRequestIsEnvelope: _UpdateMeterRequestIsEnvelope = true

// `Price` is a TypeSpec named union (`@discriminated union Price { free: ...,
// flat: ..., ... }`); before the emitter aliased named unions to a `types.ts`
// declaration, only its expansion (`PriceFree | PriceFlat | ...`) was
// reachable at each use site, so `Price` itself could not be imported,
// annotated, or narrowed by a standalone function. This narrows the imported
// alias by its discriminator and reads a variant-specific field, so a
// regression that inlines the union again (or drops the alias export) fails
// to compile.
function priceAmount(price: Price): string | number {
  switch (price.type) {
    case 'free':
      return 0
    case 'flat':
    case 'unit':
      return price.amount
    case 'graduated':
    case 'volume':
      return price.tiers.length
  }
}

describe('package-root type exports', () => {
  it('resolves formerly colliding names to the operation envelope', () => {
    expect(_updateMeterRequestIsEnvelope).toBe(true)
  })

  it('imports and narrows the named union alias Price', () => {
    const free: Price = { type: 'free' }
    const flat: Price = { type: 'flat', amount: '10' }
    expect(priceAmount(free)).toBe(0)
    expect(priceAmount(flat)).toBe('10')
  })
})
