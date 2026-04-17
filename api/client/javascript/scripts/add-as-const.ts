import { readFileSync, writeFileSync } from 'node:fs'

/**
 * Post-generation workaround for orval's zod output: object-literal defaults
 * are emitted without `as const`, so property values widen to `string` and
 * fail Zod's `.default()` signature when the schema expects a literal type.
 * Remove this script once orval emits `as const` for object-literal defaults.
 * See: https://github.com/orval-labs/orval/issues/3244
 */
const file = new URL('../src/zod/index.ts', import.meta.url)
const src = readFileSync(file, 'utf8')
const fixed = src.replace(
  /(^export const \w+Default =\s*\{[^{}]*\})/gm,
  '$1 as const',
)
writeFileSync(file, fixed)
