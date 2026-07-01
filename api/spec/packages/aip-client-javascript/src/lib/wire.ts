import type { output, ZodType } from 'zod'

export function toCamelCase(name: string): string {
  return name.replace(/_([a-z0-9])/g, (_m, c: string) => c.toUpperCase())
}

export function toSnakeCase(name: string): string {
  return name.replace(/([A-Z])/g, (_m, c: string) => `_${c.toLowerCase()}`)
}

type ZodDef = {
  type: string
  innerType?: ZodType
  valueType?: ZodType
  element?: ZodType
  discriminator?: string
  options?: ZodType[]
}

function def(schema: ZodType | undefined): ZodDef | undefined {
  return (schema as { def?: ZodDef } | undefined)?.def
}

// Unwrap optional/nullable/default to the schema that describes the value's
// shape. These wrappers never change keys, so the walker looks through them
// before classifying a node.
function unwrap(schema: ZodType | undefined): ZodType | undefined {
  let current = schema
  for (let i = 0; i < 100 && current; i++) {
    const d = def(current)
    if (
      d &&
      (d.type === 'optional' ||
        d.type === 'nullable' ||
        d.type === 'default') &&
      d.innerType
    ) {
      current = d.innerType
      continue
    }
    return current
  }
  /* v8 ignore next -- loop returns inside; reached only past the cycle guard */
  return current
}

function shapeOf(
  schema: ZodType | undefined,
): Record<string, ZodType> | undefined {
  return (schema as { shape?: Record<string, ZodType> } | undefined)?.shape
}

// The element schema for array data: the schema's own element when it is an
// array, or the array variant's element when it is a union of T and T[]
// (the single-or-batch body shape).
function arrayElement(schema: ZodType | undefined): ZodType | undefined {
  const d = def(schema)
  if (d?.type === 'array') {
    return d.element
  }
  if (d?.type === 'union') {
    for (const option of d.options ?? []) {
      const od = def(unwrap(option))
      if (od?.type === 'array') {
        return od.element
      }
    }
  }
  return undefined
}

// Whether a value schema carries renamable fields (object/record/array/union of
// such), so a record value is recursed only when it is model-shaped. Scalars,
// literals, unknown, and any are left untouched.
function hasRenamableShape(schema: ZodType | undefined): boolean {
  const s = unwrap(schema)
  const d = def(s)
  /* v8 ignore next 3 -- a record always has a value schema; defensive only */
  if (!d) {
    return false
  }
  if (d.type === 'object' || d.type === 'record') {
    return true
  }
  if (d.type === 'array') {
    return hasRenamableShape(d.element)
  }
  if (d.type === 'union') {
    return (d.options ?? []).some(hasRenamableShape)
  }
  return false
}

type Direction = {
  // The wire→public or public→wire key rename for object fields.
  rename: (key: string) => string
  // The data key holding a discriminated union's discriminator, given the
  // schema's (camelCase) discriminator key.
  discriminatorKey: (camelKey: string) => string
}

function walk(
  data: unknown,
  schema: ZodType | undefined,
  dir: Direction,
): unknown {
  if (data === null || data === undefined) {
    return data
  }
  const s = unwrap(schema)
  const d = def(s)
  if (Array.isArray(data)) {
    // The schema may be the array itself or a union with an array variant
    // (e.g. a single-or-batch body `T | T[]`); resolve the element schema from
    // whichever applies so array items are still walked with their shape.
    const element = arrayElement(s)
    return data.map((item) => walk(item, element, dir))
  }
  if (typeof data !== 'object') {
    return data
  }
  const record = data as Record<string, unknown>

  if (d?.type === 'record') {
    // Record keys are user data (label/dimension names) — preserved verbatim.
    // Only the value is walked, and only when it is model-shaped.
    const valueSchema = hasRenamableShape(d.valueType) ? d.valueType : undefined
    const out: Record<string, unknown> = {}
    for (const [key, value] of Object.entries(record)) {
      out[key] = valueSchema ? walk(value, valueSchema, dir) : value
    }
    return out
  }

  if (d?.type === 'union') {
    const variant = selectVariant(record, s, dir)
    if (!variant) {
      // No confident match: leave keys untransformed rather than guess.
      return data
    }
    return walk(data, variant, dir)
  }

  if (d?.type === 'object') {
    const shape = shapeOf(s) ?? {}
    const out: Record<string, unknown> = {}
    for (const [key, value] of Object.entries(record)) {
      const fieldSchema = fieldFor(shape, key)
      // Keys the schema does not declare are dropped, so the result matches the
      // typed shape exactly (a server-added field has no place in the type).
      if (fieldSchema === undefined) {
        continue
      }
      out[dir.rename(key)] = walk(value, fieldSchema, dir)
    }
    return out
  }

  // Scalar or unknown schema: pass through untransformed.
  return data
}

// Resolve a data key to its field schema. The schema is camelCase-keyed; a
// wire→public data key is snake, so it is camelized to index the shape.
function fieldFor(
  shape: Record<string, ZodType>,
  dataKey: string,
): ZodType | undefined {
  return shape[dataKey] ?? shape[toCamelCase(dataKey)]
}

function selectVariant(
  data: Record<string, unknown>,
  schema: ZodType | undefined,
  dir: Direction,
): ZodType | undefined {
  const d = def(schema)
  const options = d?.options ?? []
  if (d?.discriminator && schema) {
    // O(1) dispatch on the discriminator literal. The data key is the wire-name in
    // fromWire (snake) and the public name in toWire (camel); the variant map is
    // keyed by the literal value, which is identical in both directions.
    const dataKey = dir.discriminatorKey(d.discriminator)
    return variantsByDiscriminator(schema, d).get(data[dataKey])
  }
  // Non-discriminated union: the codegen gate guarantees at most one object
  // variant (it fails the build for a mapped union with two or more), so the single
  // object-shaped option is unambiguous. Other variants (scalars, arrays) reach the
  // walk through their own data-kind branches, not here.
  return options.find((option) => def(unwrap(option))?.type === 'object')
}

// Memoized literal→variant map for a discriminated union, built once per schema.
const variantMapCache = new WeakMap<ZodType, Map<unknown, ZodType>>()

function variantsByDiscriminator(
  schema: ZodType,
  d: ZodDef,
): Map<unknown, ZodType> {
  const cached = variantMapCache.get(schema)
  if (cached) {
    return cached
  }
  const map = new Map<unknown, ZodType>()
  for (const option of d.options ?? []) {
    const shape = shapeOf(unwrap(option))
    const literal = literalValue(shape?.[d.discriminator as string])
    if (literal !== undefined) {
      map.set(literal, option)
    }
  }
  variantMapCache.set(schema, map)
  return map
}

function literalValue(schema: ZodType | undefined): unknown {
  const s = unwrap(schema)
  if (def(s)?.type === 'literal') {
    return (s as { value?: unknown }).value
  }
  /* v8 ignore next -- a discriminated-union variant's discriminator is a literal */
  return undefined
}

const toWireDirection: Direction = {
  rename: toSnakeCase,
  discriminatorKey: (camelKey) => camelKey,
}

const fromWireDirection: Direction = {
  rename: toCamelCase,
  discriminatorKey: (camelKey) => toSnakeCase(camelKey),
}

// Rewrite a request body or query object from the camelCase public shape to the
// snake_case wire shape, driven by its schema. Record keys (label/dimension names)
// are preserved. The return is typed as the input `T` so call sites stay cast-free
// (the runtime object has snake keys, but the value is write-only — it flows
// straight into `json:`/`toURLSearchParams`, both of which accept any object).
export function toWire<T>(data: T, schema: ZodType): T {
  return walk(data, schema, toWireDirection) as T
}

// Rewrite a response body from the snake_case wire shape to the camelCase public
// shape. Renames keys only — never coerces values or applies defaults. The result
// is the schema's output shape: `walk` produces exactly the schema's known fields
// in camelCase, so the inferred `output<S>` type describes the runtime value (the
// same wire-trust boundary as a plain `.json<T>()`, with no `.parse()`).
export function fromWire<S extends ZodType>(
  data: unknown,
  schema: S,
): output<S> {
  return walk(data, schema, fromWireDirection) as output<S>
}

// Thrown by assertValid when the optional `validate` client option is on and data
// fails its schema. request() catches it like any Error and surfaces it as
// Result.error.
export class ValidationError extends Error {
  constructor(
    message: string,
    public readonly issues: unknown,
  ) {
    super(message)
    this.name = 'ValidationError'
  }
}

// Opt-in schema check used by the funcs (when the validate option is on) against
// the snake_case wire payload: the request body after toWire, the raw response
// before fromWire, each against its generated `…Wire` schema. It is a GATE, not a
// transform — the safeParse output (coercions/defaults) is discarded, so validation
// never mutates the payload or return value. Off by default; the SDK does not
// validate by default (additive server fields must not break clients).
export function assertValid(schema: ZodType, data: unknown): void {
  const result = schema.safeParse(data)
  if (!result.success) {
    throw new ValidationError('schema validation failed', result.error.issues)
  }
}
