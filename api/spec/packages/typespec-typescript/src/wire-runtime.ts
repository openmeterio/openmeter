import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

/**
 * The boundary mapper source, emitted verbatim as `src/lib/wire.ts` in the
 * generated SDK. Authored as a real module under `runtime/` (type-checked and
 * unit-tested in this package) and read here so the emitted file stays in lockstep
 * with what the tests exercise.
 */
export const WIRE_RUNTIME = readFileSync(
  fileURLToPath(new URL('../src/runtime/wire.ts', import.meta.url)),
  'utf8',
)
