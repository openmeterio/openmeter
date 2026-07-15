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
  entries?: Record<string, string>
  defaultValue?: unknown
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

// The default to write for a field the caller omitted, present only when the
// field is required-with-default in the spec — emitted as `.default(x)` with no
// `.optional()` in the wrapper chain (TypeSpec `field: T = x`, not
// `field?: T = x`). Such fields must be present on the wire (e.g. a CloudEvents
// envelope's `specversion`, which the server's parser rejects when absent), yet
// the generated request type allows omitting them so callers get the documented
// default for free — toWire is the layer that has to reconcile the two.
// Spec-optional fields (`.optional().default(x)`) return undefined and stay off
// the wire: their default is the server's to apply, and materializing them
// client-side would silently overwrite server state on update requests.
function requiredDefault(
  schema: ZodType | undefined,
): { value: unknown } | undefined {
  let current = schema
  let found: { value: unknown } | undefined
  for (let i = 0; i < 100 && current; i++) {
    const d = def(current)
    /* v8 ignore next 3 -- every zod schema carries a def; guards the type only */
    if (!d) {
      break
    }
    if (d.type === 'optional') {
      return undefined
    }
    if (d.type === 'default') {
      found ??= { value: d.defaultValue }
    } else if (d.type !== 'nullable') {
      break
    }
    current = d.innerType
  }
  return found
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

// Whether a record value schema needs walking: it carries renamable fields
// (object/record/array/union of such) or date-typed values that must map
// between `Date` and the RFC 3339 wire string. Other scalars, literals,
// unknown, and any are left untouched, so user data (labels, dimensions,
// event payloads) is never rewritten.
function needsWalk(schema: ZodType | undefined): boolean {
  const s = unwrap(schema)
  const d = def(s)
  /* v8 ignore next 3 -- a record always has a value schema; defensive only */
  if (!d) {
    return false
  }
  if (
    d.type === 'object' ||
    d.type === 'record' ||
    d.type === 'date' ||
    d.type === 'bigint'
  ) {
    return true
  }
  if (d.type === 'array') {
    return needsWalk(d.element)
  }
  if (d.type === 'union') {
    return (d.options ?? []).some(needsWalk)
  }
  return false
}

// Thrown by toWire when a bigint value (an int64/uint64 field) is outside
// JSON's exactly-representable integer range. The wire carries int64 as a JSON
// number (the server decodes it into a Go int64), so a value beyond 2^53-1
// cannot be sent without silent precision corruption — a typed, immediate
// failure is the only honest option. request() catches it like any Error and
// surfaces it as Result.error.
export class UnsafeIntegerError extends Error {
  constructor(value: bigint) {
    super(
      `bigint value ${value} exceeds JSON's safe integer range and cannot be sent without precision loss`,
    )
    this.name = 'UnsafeIntegerError'
  }
}

// An RFC 3339 string standing in for a `Date` on request input. The
// `Record<never, never>` intersection keeps a plain string assignable while
// stopping the union simplifier from absorbing sibling string literals — a
// bare `| string` would collapse `'immediate' | 'next_billing_cycle' | Date |
// string` to `string | Date` and kill literal autocomplete.
export type DateString = string & Record<never, never>

// The request-side widening of a payload type: every `Date` also accepts its
// RFC 3339 string form. Applied to the generated `…Request` aliases only —
// domain interfaces and response types stay `Date`, so responses always carry
// real `Date`s. At runtime the mapper passes request strings through verbatim
// (never re-parses or normalizes them), so the wire sees exactly what was given
// and the optional wire validation still checks the string against the RFC 3339
// wire schema.
export type AcceptDateStrings<T> = T extends Date
  ? Date | DateString
  : T extends (infer E)[]
    ? AcceptDateStrings<E>[]
    : T extends object
      ? { [K in keyof T]: AcceptDateStrings<T[K]> }
      : T

// Literal siblings of `Date` must survive the widening. Checked at compile
// time in both the emitter build and the generated SDK's typecheck, so a
// regression to a bare `| string` arm fails the build.
type _LiteralsSurviveWidening =
  'x' extends Extract<AcceptDateStrings<'x' | Date>, 'x'>
    ? true
    : {
        __error: 'AcceptDateStrings absorbed literal union members into string'
      }
const _literalsSurviveWidening: _LiteralsSurviveWidening = true
void _literalsSurviveWidening

type Direction = {
  // The wire→public or public→wire key rename for object fields.
  rename: (key: string) => string
  // The data key holding a discriminated union's discriminator, given the
  // schema's (camelCase) discriminator key.
  discriminatorKey: (camelKey: string) => string
  // The value mapping at a date-typed node: public `Date` → RFC 3339 wire
  // string, wire string → `Date`. Values already in the target form (or not
  // convertible) pass through unchanged.
  mapDate: (value: unknown) => unknown
  // The value mapping at a bigint-typed (int64/uint64) node: public `bigint` →
  // JSON number (throwing UnsafeIntegerError beyond 2^53-1, where JSON numbers
  // lose integer precision), wire number → `bigint`. Without the public→wire
  // mapping, JSON.stringify throws an opaque TypeError on any bigint. Values
  // already in the target form (or not convertible) pass through unchanged.
  mapBigInt: (value: unknown) => unknown
  // Whether absent required-with-default fields are materialized (see
  // requiredDefault). True only public→wire: requests must satisfy the wire
  // contract, while responses are reported as the server sent them — fromWire
  // fabricating fields would mask genuine contract violations.
  applyDefaults: boolean
  // JSON.stringify omits object properties and record entries whose value is
  // undefined. True only public→wire so validation sees the effective JSON
  // payload instead of an intermediate object that the transport cannot send.
  // Array entries are deliberately unaffected: JSON serializes undefined array
  // values as null rather than omitting them.
  omitUndefinedObjectEntries: boolean
}

// A handful of schemas are genuinely self-referential (e.g. the `and`/`or`
// legs of a filter tree), so nesting depth is bounded only by the DATA the
// server sends, not by the schema. Without a limit, a crafted or
// accidentally-deep response recurses until the JS engine throws a raw
// `RangeError: Maximum call stack size exceeded` — still caught by request()
// and surfaced as Result.error, but as an opaque native error instead of a
// typed one. 500 levels is far beyond any real filter/record/array nesting
// in the API today; it exists to fail predictably, not to constrain valid data.
const MAX_WALK_DEPTH = 500

export class DepthLimitExceededError extends Error {
  constructor() {
    super(`wire mapping exceeded maximum nesting depth (${MAX_WALK_DEPTH})`)
    this.name = 'DepthLimitExceededError'
  }
}

function walk(
  data: unknown,
  schema: ZodType | undefined,
  dir: Direction,
  depth = 0,
): unknown {
  if (data === null || data === undefined) {
    return data
  }
  // A Date can only ever mean its wire serialization, wherever it sits — a
  // typed date field, a record value, or an unknown-schema position. Wire→
  // public data never contains Date instances (it comes from JSON.parse), so
  // this only rewrites public→wire. The same holds for bigint: JSON.parse
  // never produces one, and public→wire it must become a JSON number wherever
  // it sits.
  if (data instanceof Date) {
    return dir.mapDate(data)
  }
  if (typeof data === 'bigint') {
    return dir.mapBigInt(data)
  }
  if (depth > MAX_WALK_DEPTH) {
    throw new DepthLimitExceededError()
  }
  const s = unwrap(schema)
  const d = def(s)
  // A date-typed node maps between the public `Date` and the RFC 3339 wire
  // string (fromWire revives the string; a string handed to toWire by an
  // untyped caller passes through as-is).
  if (d?.type === 'date') {
    return dir.mapDate(data)
  }
  // A bigint-typed node revives the wire's JSON number into the public
  // `bigint` (public→wire bigints were already mapped by the value check
  // above, so only fromWire reaches a number here).
  if (d?.type === 'bigint') {
    return dir.mapBigInt(data)
  }
  if (Array.isArray(data)) {
    // The schema may be the array itself or a union with an array variant
    // (e.g. a single-or-batch body `T | T[]`); resolve the element schema from
    // whichever applies so array items are still walked with their shape.
    const element = arrayElement(s)
    return data.map((item) => walk(item, element, dir, depth + 1))
  }
  if (typeof data !== 'object') {
    // A wire datetime can sit behind a union (`DateTime | null`,
    // enum-or-DateTime): revive the string only when the union's date variant
    // is its sole plausible owner, so enum literals and plain-string variants
    // pass through untouched.
    if (
      typeof data === 'string' &&
      d?.type === 'union' &&
      unionDateClaims(s, data)
    ) {
      return dir.mapDate(data)
    }
    return data
  }
  const record = data as Record<string, unknown>

  if (d?.type === 'record') {
    // Record keys are user data (label/dimension names) — preserved verbatim.
    // Only the value is walked, and only when it needs mapping. A null
    // prototype avoids the `__proto__` key silently reassigning `out`'s own
    // prototype instead of becoming a visible entry (user data may contain
    // any key, including reserved object-literal property names). The
    // prototype is restored once every key is a plain own property, so the
    // returned object still behaves normally for consumers (instanceof,
    // template literals) — `Object.prototype` itself was never touched.
    const valueSchema = needsWalk(d.valueType) ? d.valueType : undefined
    const out: Record<string, unknown> = Object.create(null)
    for (const [key, value] of Object.entries(record)) {
      if (dir.omitUndefinedObjectEntries && value === undefined) {
        continue
      }
      out[key] = valueSchema ? walk(value, valueSchema, dir, depth + 1) : value
    }
    Object.setPrototypeOf(out, Object.prototype)
    return out
  }

  if (d?.type === 'union') {
    const variant = selectVariant(record, s, dir)
    if (!variant) {
      // No confident match: leave keys untransformed rather than guess.
      return data
    }
    return walk(data, variant, dir, depth + 1)
  }

  if (d?.type === 'object') {
    const shape = shapeOf(s) ?? {}
    // A null prototype avoids two failure modes from data-controlled keys
    // like `__proto__`/`constructor`: (1) `fieldFor` below reading an
    // inherited Object.prototype member instead of correctly treating the
    // key as schema-undeclared, and (2) the assignment at the end of this
    // loop reassigning `out`'s own prototype instead of adding a visible key.
    const out: Record<string, unknown> = Object.create(null)
    for (const [key, value] of Object.entries(record)) {
      if (dir.omitUndefinedObjectEntries && value === undefined) {
        continue
      }
      const fieldSchema = fieldFor(shape, key)
      // Keys the schema does not declare are dropped, so the result matches the
      // typed shape exactly (a server-added field has no place in the type).
      if (fieldSchema === undefined) {
        continue
      }
      out[dir.rename(key)] = walk(value, fieldSchema, dir, depth + 1)
    }
    if (dir.applyDefaults) {
      // Shape keys are generated camelCase identifiers (never data-controlled),
      // so direct indexing into `record` is safe here. Runs after the data loop
      // so an explicit `key: undefined` entry is also replaced by the default.
      for (const [key, fieldSchema] of Object.entries(shape)) {
        if (record[key] !== undefined) {
          continue
        }
        const dflt = requiredDefault(fieldSchema)
        if (dflt !== undefined) {
          out[dir.rename(key)] = walk(dflt.value, fieldSchema, dir, depth + 1)
        }
      }
    }
    Object.setPrototypeOf(out, Object.prototype)
    return out
  }

  // Scalar or unknown schema: pass through untransformed.
  return data
}

// Resolve a data key to its field schema. The schema is camelCase-keyed; a
// wire→public data key is snake, so it is camelized to index the shape.
// Own-property checks (not `shape[key]`) so a data-controlled key like
// `__proto__` or `constructor` cannot resolve to an inherited
// Object.prototype member and be mistaken for a declared schema field.
function fieldFor(
  shape: Record<string, ZodType>,
  dataKey: string,
): ZodType | undefined {
  if (Object.hasOwn(shape, dataKey)) {
    return shape[dataKey]
  }
  const camelKey = toCamelCase(dataKey)
  return Object.hasOwn(shape, camelKey) ? shape[camelKey] : undefined
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

// Whether a union's date variant is the sole plausible owner of a string
// value: the union carries a date option, no string-capable sibling (a plain
// string variant, an enum containing the value, an equal string literal)
// claims it, and the value actually parses as a date. `DateTime | null`
// revives its RFC 3339 string; `'immediate' | DateTime` keeps the enum
// literal a string. Fail-open: an unclaimed string stays a string.
function unionDateClaims(schema: ZodType | undefined, value: string): boolean {
  let hasDate = false
  for (const option of def(schema)?.options ?? []) {
    const od = def(unwrap(option))
    if (od?.type === 'date') {
      hasDate = true
    } else if (od?.type === 'string') {
      return false
    } else if (
      od?.type === 'enum' &&
      Object.values(od.entries ?? {}).includes(value)
    ) {
      return false
    } else if (od?.type === 'literal' && literalValue(option) === value) {
      return false
    }
  }
  return hasDate && !Number.isNaN(Date.parse(value))
}

const toWireDirection: Direction = {
  rename: toSnakeCase,
  discriminatorKey: (camelKey) => camelKey,
  mapDate: (value) => (value instanceof Date ? value.toISOString() : value),
  mapBigInt: (value) => {
    if (typeof value !== 'bigint') {
      return value
    }
    if (
      value > BigInt(Number.MAX_SAFE_INTEGER) ||
      value < -BigInt(Number.MAX_SAFE_INTEGER)
    ) {
      throw new UnsafeIntegerError(value)
    }
    return Number(value)
  },
  applyDefaults: true,
  omitUndefinedObjectEntries: true,
}

const fromWireDirection: Direction = {
  rename: toCamelCase,
  discriminatorKey: (camelKey) => toSnakeCase(camelKey),
  mapDate: (value) => (typeof value === 'string' ? new Date(value) : value),
  mapBigInt: (value) =>
    typeof value === 'number' && Number.isInteger(value)
      ? BigInt(value)
      : value,
  applyDefaults: false,
  omitUndefinedObjectEntries: false,
}

// Rewrite a request body or query object from the camelCase public shape to the
// snake_case wire shape, driven by its schema. Record keys (label/dimension names)
// are preserved; `Date` values serialize to RFC 3339 strings; `bigint` values
// (int64 fields) become JSON numbers; omitted required-with-default fields are
// filled with their declared default (see requiredDefault); explicit undefined
// object/record entries are omitted just as JSON.stringify would omit them. The
// return is typed as the input `T` so call sites stay cast-free (the runtime object
// has snake keys and wire-encoded dates, but the value is write-only — it flows
// straight into `json:`/`toURLSearchParams`, both of which accept any object).
export function toWire<T>(data: T, schema: ZodType): T {
  return walk(data, schema, toWireDirection) as T
}

// Rewrite a response body from the snake_case wire shape to the camelCase public
// shape: renames keys and revives RFC 3339 strings into `Date`s at date-typed
// nodes — never applies defaults or any other coercion. The result is the
// schema's output shape: `walk` produces exactly the schema's known fields in
// camelCase, so the inferred `output<S>` type describes the runtime value (the
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
