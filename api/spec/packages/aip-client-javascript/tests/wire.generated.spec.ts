import { describe, expect, it } from 'vitest'
import * as schemas from '../src/models/schemas.js'
import { fromWire, toWire } from '../src/lib/wire.js'
import {
  collectFieldKeys,
  operationSchemaPairs,
  sampleCamel,
  sampleSnake,
} from './wire-helpers.js'

const SNAKE_KEY = /_/
const CAMEL_KEY = /[A-Z]/

const pairs = operationSchemaPairs(schemas as Record<string, unknown>)

// The casing-leak invariant, applied to every operation: toWire produces a
// snake_case body (no camelCase field key escapes), fromWire produces a camelCase
// shape (no snake_case field key escapes). Preserved record user keys
// (`user_key_*`) are excluded by collectFieldKeys, so a legitimately-preserved
// key never trips the assertion. Field names that are single-word in both cases
// pass trivially.
describe('per-operation wire casing (toWire → snake, fromWire → camel)', () => {
  for (const { base, body, response } of pairs) {
    if (body) {
      it(`${base}: request body has no camelCase field key on the wire`, () => {
        const wire = toWire(sampleCamel(body), body)
        const leaked = collectFieldKeys(wire).filter((k) => CAMEL_KEY.test(k))
        expect(leaked).toEqual([])
      })

      // Round-trips the public sample through the wire and back. Catches a silent
      // drop or a mis-mapped name that the no-leak check alone cannot see: the
      // sample contains only in-schema fields, so nothing should be dropped and
      // identity must hold exactly.
      it(`${base}: request body round-trips (camel → wire → camel)`, () => {
        const sample = sampleCamel(body)
        expect(fromWire(toWire(sample, body), body)).toEqual(sample)
      })
    }
    if (response) {
      it(`${base}: response has no snake_case field key after mapping`, () => {
        const camel = fromWire(sampleSnake(response), response)
        const leaked = collectFieldKeys(camel).filter((k) => SNAKE_KEY.test(k))
        expect(leaked).toEqual([])
      })

      it(`${base}: response round-trips (wire → camel → wire)`, () => {
        const wire = sampleSnake(response)
        expect(toWire(fromWire(wire, response), response)).toEqual(wire)
      })
    }
  }
})

// The strict `…Wire` schemas behind the validate option are emitted from the same
// TypeSpec walk as the camelCase schemas (one pass parameterized by casing +
// strictness), so they agree with the data mapper by construction — no per-op
// runtime sweep is needed. wire.spec.ts spot-checks the generated wire schemas
// (strict closed model, open record-spread, discriminator, record value, cyclic).
