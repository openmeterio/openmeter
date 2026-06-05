import {
  type Enum,
  type IntrinsicType,
  type LiteralType,
  type Model,
  type Program,
  type Scalar,
  type Tuple,
  type Type,
  type Union,
} from '@typespec/compiler'
import { $ } from '@typespec/compiler/typekit'
import type { Typekit } from '@typespec/compiler/typekit'
import { bodyProperties, isRecord } from './utils.jsx'

/**
 * A reference resolver: returns the TypeScript interface name for a type that is
 * emitted as a named interface, or `undefined` when the type must be inlined.
 */
export type RefName = (type: Type) => string | undefined

/**
 * Whether the walked type expresses a request (`input`) or response (`output`)
 * shape. Leaf types are identical; only optionality of anonymous objects differs
 * — a defaulted field is optional on input and required on output.
 */
export type IoMode = 'input' | 'output'

/** True when a property is optional in the given direction. */
export function isOptional(
  prop: { optional: boolean; defaultValue?: unknown },
  io: IoMode,
): boolean {
  if (io === 'input') {
    return prop.optional || prop.defaultValue !== undefined
  }
  return prop.optional && prop.defaultValue === undefined
}

/**
 * Maps a TypeSpec type to a concrete TypeScript type string, mirroring the
 * inferred output of {@link zodBaseSchemaParts} so the emitted type is mutually
 * assignable with `z.output<typeof schema>` (enforced by the conformance guard).
 *
 * Named models resolve to their interface via {@link RefName}; everything else
 * is inlined. Leaf scalars follow the same wire-native decisions as the zod
 * emitter (dates and durations are strings, 64-bit integers are `bigint`).
 */
export function tsTypeOf(
  program: Program,
  type: Type,
  refName: RefName,
  io: IoMode = 'output',
): string {
  const tk = $(program)
  switch (type.kind) {
    case 'ModelProperty':
      return tsTypeOf(program, type.type, refName, io)
    case 'Scalar':
      return scalarType(tk, type)
    case 'String':
    case 'Number':
    case 'Boolean':
      return literalType(type)
    case 'Model':
      return modelType(program, type, refName, io)
    case 'Union':
      return unionType(program, type, refName, io)
    case 'Enum':
      return enumType(type)
    case 'EnumMember':
      return JSON.stringify(type.value ?? type.name)
    case 'Tuple':
      return tupleType(program, type, refName, io)
    case 'Intrinsic':
      return intrinsicType(type)
    default:
      return 'unknown'
  }
}

function literalType(type: LiteralType): string {
  switch (type.kind) {
    case 'String':
      return JSON.stringify(type.value)
    case 'Number':
    case 'Boolean':
      return `${type.value}`
  }
}

function intrinsicType(type: IntrinsicType): string {
  switch (type.name) {
    case 'null':
      return 'null'
    case 'never':
      return 'never'
    case 'void':
      return 'void'
    default:
      return 'unknown'
  }
}

function scalarType(tk: Typekit, type: Scalar): string {
  if (tk.scalar.extendsBoolean(type)) {
    return 'boolean'
  }
  if (tk.scalar.extendsNumeric(type)) {
    if (
      tk.scalar.extendsInteger(type) &&
      (tk.scalar.extendsInt64(type) || tk.scalar.extendsUint64(type))
    ) {
      return 'bigint'
    }
    return 'number'
  }
  if (tk.scalar.extendsString(type)) {
    return 'string'
  }
  if (tk.scalar.extendsPlainTime(type) || tk.scalar.extendsDuration(type)) {
    return 'string'
  }
  if (
    tk.scalar.extendsPlainDate(type) ||
    tk.scalar.extendsUtcDateTime(type) ||
    tk.scalar.extendsOffsetDateTime(type)
  ) {
    return dateScalarType(tk, type)
  }
  return 'unknown'
}

/**
 * Date/time scalars are wire-native strings unless an encoding maps them onto a
 * numeric base (e.g. `unixTimestamp`), matching the zod emitter's choice of
 * `z.string().datetime()` for RFC 3339 over `z.coerce.date()`.
 */
function dateScalarType(tk: Typekit, type: Scalar): string {
  const encoding = tk.scalar.getEncoding(type)
  if (encoding === undefined || encoding.encoding === 'rfc3339') {
    return 'string'
  }
  if (encoding.type.kind === 'Scalar') {
    return scalarType(tk, encoding.type)
  }
  return 'string'
}

function enumType(type: Enum): string {
  const members = [...type.members.values()].map((member) =>
    JSON.stringify(member.value ?? member.name),
  )
  return members.length > 0 ? members.join(' | ') : 'never'
}

function tupleType(
  program: Program,
  type: Tuple,
  refName: RefName,
  io: IoMode,
): string {
  const parts = type.values.map((value) =>
    tsTypeOf(program, value, refName, io),
  )
  return `[${parts.join(', ')}]`
}

function unionType(
  program: Program,
  type: Union,
  refName: RefName,
  io: IoMode,
): string {
  const variants = [...type.variants.values()].map((variant) =>
    tsTypeOf(program, variant.type, refName, io),
  )
  return variants.length > 0 ? variants.join(' | ') : 'never'
}

function modelType(
  program: Program,
  type: Model,
  refName: RefName,
  io: IoMode,
): string {
  const tk = $(program)

  if (tk.array.is(type) && type.indexer) {
    const element = tsTypeOf(program, type.indexer.value, refName, io)
    return needsParensInArray(element) ? `(${element})[]` : `${element}[]`
  }

  // A named model (including a named record like `Labels`) refs its interface so
  // its docs stay reachable; only anonymous shapes are inlined structurally.
  const named = refName(type)
  if (named) {
    return named
  }

  if (isRecord(program, type) && type.indexer) {
    const value = tsTypeOf(program, type.indexer.value, refName, io)
    return `Record<string, ${value}>`
  }

  return anonymousObjectType(program, type, refName, io)
}

function anonymousObjectType(
  program: Program,
  type: Model,
  refName: RefName,
  io: IoMode,
): string {
  const fields = bodyProperties(program, type).map((prop) => {
    const optional = isOptional(prop, io) ? '?' : ''
    return `${prop.name}${optional}: ${tsTypeOf(program, prop.type, refName, io)}`
  })
  if (type.baseModel) {
    const base = refName(type.baseModel)
    if (base && fields.length === 0) {
      return base
    }
  }
  return fields.length > 0
    ? `{ ${fields.join('; ')} }`
    : 'Record<string, never>'
}

function needsParensInArray(element: string): boolean {
  return element.includes('|') && !element.startsWith('(')
}
