import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

/**
 * Post-generation workaround for orval's zod output: object-literal defaults
 * are emitted without `as const`, so property values widen to `string` and
 * fail Zod's `.default()` signature when the schema expects a literal type.
 * Remove this script once orval emits `as const` for object-literal defaults.
 * See: https://github.com/orval-labs/orval/issues/3244
 *
 * Optional first CLI arg is the target file (resolved from cwd), used by the v3
 * shim to point at src/v3/zod/index.ts. Defaults to the v1 zod output.
 */
const file =
  process.argv[2] ??
  fileURLToPath(new URL('../src/zod/index.ts', import.meta.url))
const src = readFileSync(file, 'utf8')
const fixed = src.replace(
  /(^export const \w+Default =\s*\{[^{}]*\})/gm,
  '$1 as const',
)
writeFileSync(file, fixed)
