import type { ZodType } from 'zod'

type ZodDef = {
  type: string
  innerType?: ZodType
  valueType?: ZodType
  element?: ZodType
  options?: ZodType[]
  entries?: Record<string, string>
}

function def(schema: ZodType | undefined): ZodDef | undefined {
  return (schema as { def?: ZodDef } | undefined)?.def
}

function shapeOf(
  schema: ZodType | undefined,
): Record<string, ZodType> | undefined {
  return (schema as { shape?: Record<string, ZodType> } | undefined)?.shape
}

/**
 * A value satisfying a schema's structure, with camelCase object keys (the emitted
 * schemas are camelCase) and a deliberately underscore-containing record key so the
 * casing assertion can tell a leaked field key from a preserved user key.
 *
 * Every field whose sample is defined is populated (so optional camelCase fields are
 * exercised). A key whose sample comes back undefined is omitted rather than set to
 * undefined, so a discriminated-union variant always carries its discriminator
 * literal and the mapper can resolve every variant it descends into. The depth cap
 * bounds the self-referential filter schemas.
 */
export function sampleCamel(schema: ZodType | undefined, depth = 0): unknown {
  const d = def(schema)
  if (!d || depth > 12) {
    return undefined
  }
  switch (d.type) {
    case 'object': {
      const out: Record<string, unknown> = {}
      for (const [k, v] of Object.entries(shapeOf(schema) ?? {})) {
        const value = sampleCamel(v, depth + 1)
        if (value !== undefined) {
          out[k] = value
        }
      }
      return out
    }
    case 'optional':
    case 'nullable':
    case 'default':
      return sampleCamel(d.innerType, depth)
    case 'array':
      return [sampleCamel(d.element, depth + 1)]
    case 'record':
      return { user_key_a: sampleCamel(d.valueType, depth + 1) }
    case 'union':
      return sampleCamel(d.options?.[0], depth)
    case 'literal':
      return (schema as { value?: unknown }).value
    case 'enum':
      // Without this, enum fields fall out of every sample entirely — and a
      // required-with-default enum (e.g. an invoice line's `category`) then
      // breaks response round-trip identity, because toWire materializes the
      // default the sample never carried.
      return Object.values(d.entries ?? {})[0]
    case 'string':
      return 's'
    case 'int':
    case 'number':
      return 1
    case 'bigint':
      return 1n
    case 'boolean':
      return true
    case 'date':
      return new Date(0)
    default:
      return undefined
  }
}

// Snake-cased counterpart for response samples: object keys are snake_cased so
// fromWire has wire-shaped input. Record user keys and literal values are kept.
export function sampleSnake(schema: ZodType | undefined, depth = 0): unknown {
  const value = sampleCamel(schema, depth)
  return toSnakeKeys(value)
}

function toSnakeKeys(value: unknown): unknown {
  if (value instanceof Date) {
    // Wire samples carry the RFC 3339 string a JSON payload would.
    return value.toISOString()
  }
  if (typeof value === 'bigint') {
    // Wire samples carry the JSON number an int64 payload would.
    return Number(value)
  }
  if (Array.isArray(value)) {
    return value.map(toSnakeKeys)
  }
  if (value && typeof value === 'object') {
    const out: Record<string, unknown> = {}
    for (const [k, v] of Object.entries(value)) {
      const key = k.startsWith('user_key_')
        ? k
        : k.replace(/([A-Z])/g, (_m, c: string) => `_${c.toLowerCase()}`)
      out[key] = toSnakeKeys(v)
    }
    return out
  }
  return value
}

/**
 * Every object key in the value, except keys of a `user_key_*` map (preserved
 * record keys) and date values. Used to assert no casing leaks past the mapper.
 */
export function collectFieldKeys(
  value: unknown,
  keys: string[] = [],
): string[] {
  if (Array.isArray(value)) {
    for (const item of value) {
      collectFieldKeys(item, keys)
    }
    return keys
  }
  if (value && typeof value === 'object' && !(value instanceof Date)) {
    for (const [k, v] of Object.entries(value)) {
      if (k.startsWith('user_key_')) {
        // Preserved user record key — not itself a schema field, so it's excluded
        // from the leak check. Its value can still be model-shaped (e.g. a
        // governance feature-access record value), so keep walking into it —
        // skipping the value too would blind the leak check to casing bugs in
        // any schema field nested under a record.
        collectFieldKeys(v, keys)
        continue
      }
      keys.push(k)
      collectFieldKeys(v, keys)
    }
  }
  return keys
}

/** Per-op schemas grouped by base name, from the generated schemas module. */
export function operationSchemaPairs(
  schemas: Record<string, unknown>,
): Array<{ base: string; body?: ZodType; response?: ZodType }> {
  const bases = new Map<string, { body?: ZodType; response?: ZodType }>()
  for (const [name, schema] of Object.entries(schemas)) {
    const bodyMatch = name.match(/^(.*)Body$/)
    const responseMatch = name.match(/^(.*)Response$/)
    if (bodyMatch) {
      const entry = bases.get(bodyMatch[1]) ?? {}
      entry.body = schema as ZodType
      bases.set(bodyMatch[1], entry)
    } else if (responseMatch) {
      const entry = bases.get(responseMatch[1]) ?? {}
      entry.response = schema as ZodType
      bases.set(responseMatch[1], entry)
    }
  }
  return [...bases.entries()].map(([base, v]) => ({ base, ...v }))
}
