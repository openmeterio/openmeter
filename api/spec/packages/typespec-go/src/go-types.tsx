import * as ay from '@alloy-js/core'
import * as go from '@alloy-js/go'
import {
  getFriendlyName,
  resolveEncodedName,
  walkPropertiesInherited,
  type Model,
  type ModelProperty,
  type Program,
  type Scalar,
  type Type,
} from '@typespec/compiler'
import { $ } from '@typespec/compiler/typekit'
import { isHeader, isStatusCode, isVisible, Visibility } from '@typespec/http'

const stripNamePrefixes = new WeakMap<Program, readonly string[]>()
const resolvedTypeNames = new WeakMap<Program, Map<string, string>>()
const syntheticTypeNames = new WeakMap<Program, Map<Type, string>>()
const projectionsByProgram = new WeakMap<Program, GoProjections>()

export function inputVariantName(name: string): string {
  return `${name}Input`
}

/** Which payload projection of a model a declaration or reference lives in:
 * read models come from response bodies, input models from request bodies. */
export type GoPayloadMode = 'read' | 'input'

export interface GoDeclarationPlan {
  name: string
  mode: GoPayloadMode
}

/**
 * Payload-context visibility state for one emitted program.
 *
 * Response-reachable models are rendered in a read context, so their
 * Create-/Update-only properties (which the server never returns) are dropped.
 * Request-reachable models are rendered in an input context that keeps every
 * spec-projected property. A model needed by both contexts with different
 * shapes is emitted twice (Name plus NameInput); `declarations` records
 * exactly which projections each type emits, and `aliases` redirects
 * declarations that would duplicate a structurally identical canonical type.
 */
export interface GoProjections {
  readReachable: Set<Type>
  inputReachable: Set<Type>
  divergent: Set<Type>
  aliases: Map<string, string>
  declarations: Map<Type, GoDeclarationPlan[]>
}

export function configureGoProjections(
  program: Program,
  projections: GoProjections,
): void {
  projectionsByProgram.set(program, projections)
}

export function goProjectionsOf(program: Program): GoProjections | undefined {
  return projectionsByProgram.get(program)
}

export function setSyntheticTypeNames(
  program: Program,
  names: Map<Type, string>,
): void {
  syntheticTypeNames.set(program, names)
}

/**
 * The Go type name a field or accessor must reference for `type` in the given
 * payload context: the input twin when the type emits one, then any
 * structural-dedupe alias pointing at the canonical declaration.
 */
export function goReferenceTypeName(
  program: Program,
  type: Type,
  name: string,
  mode?: 'input' | 'output',
): string {
  const projections = goProjectionsOf(program)
  if (!projections) {
    return name
  }

  let final =
    mode === 'input' && projections.divergent.has(type)
      ? inputVariantName(name)
      : name
  while (projections.aliases.has(final)) {
    final = projections.aliases.get(final)!
  }
  return final
}

export interface GoField {
  name: string
  wireName: string
  type: ay.Children
  /** Plain-text rendering of `type` when expressible without alloy context;
   * consumed by the structural-dedupe signature. */
  typeText?: string
  optional: boolean
  nilable: boolean
  doc?: string
}

export type GoQueryScalarKind =
  | 'string'
  | 'boolean'
  | 'integer'
  | 'float'
  | 'dateTime'

export type GoQueryFilterKind =
  | 'string'
  | 'stringExact'
  | 'dateTime'
  | 'numeric'
  | 'boolean'
  | 'labels'
  | 'scalar'

export interface GoTypeOptions {
  mode?: 'input' | 'output'
}

export interface GoTypeResult {
  type: ay.Children
  /** Plain-text rendering of `type`; undefined only for inline struct
   * literals, which cannot participate in structural dedupe. */
  text?: string
  nilable: boolean
  jsonNullable?: boolean
}

export function configureGoTypeNames(
  program: Program,
  prefixes: readonly string[],
): void {
  stripNamePrefixes.set(program, prefixes)
  resolvedTypeNames.delete(program)
}

export function resolveGoTypeNames(program: Program, types: Iterable<Type>) {
  const baseNames = [...types]
    .map((type) => baseGoTypeName(program, type))
    .filter((name): name is string => name !== undefined)

  resolvedTypeNames.set(
    program,
    resolveStrippedNames(baseNames, stripNamePrefixes.get(program) ?? []),
  )
}

export function goFields(
  program: Program,
  model: Model,
  options: {
    omit?: Set<string>
    mode?: 'input' | 'output'
  } = {},
): GoField[] {
  const fields: GoField[] = []
  const projections = goProjectionsOf(program)
  // Payload-context visibility: a response-reachable model is rendered in a
  // read context, so Create-/Update-only properties (never returned by the
  // server) are dropped from it. Request payloads render in input mode and
  // keep every property the spec projected into them; without a configured
  // projection registry (unit tests) nothing is filtered.
  const filterRead =
    options.mode !== 'input' &&
    projections !== undefined &&
    projections.readReachable.has(model)

  for (const property of walkPropertiesInherited(model)) {
    if (
      isStatusCode(program, property) ||
      isHeader(program, property) ||
      options.omit?.has(property.name)
    ) {
      continue
    }
    if (filterRead && !isVisible(program, property, Visibility.Read)) {
      continue
    }

    const mapped = goType(program, property.type, options)
    const optional =
      property.optional ||
      (options.mode === 'input' && property.defaultValue !== undefined)
    const pointerOptional =
      optional &&
      !mapped.jsonNullable &&
      (!mapped.nilable || options.mode === 'input')
    fields.push({
      name: goExportedName(property.name),
      wireName: resolveEncodedName(
        program,
        property as ModelProperty & { name: string },
        'application/json',
      ),
      type: pointerOptional ? (
        <go.Pointer>{mapped.type}</go.Pointer>
      ) : (
        mapped.type
      ),
      typeText:
        mapped.text === undefined
          ? undefined
          : `${pointerOptional ? '*' : ''}${mapped.text}`,
      optional,
      nilable: mapped.nilable,
      doc: $(program).type.getDoc(property),
    })
  }

  return fields
}

export function goType(
  program: Program,
  type: Type,
  options: GoTypeOptions = {},
): GoTypeResult {
  switch (type.kind) {
    case 'Boolean':
      return { type: 'bool', text: 'bool', nilable: false }
    case 'Number': {
      const text = Number.isInteger(type.value) ? 'int' : 'float64'
      return { type: text, text, nilable: false }
    }
    case 'String':
      return { type: 'string', text: 'string', nilable: false }
    case 'Scalar':
      return scalarType(type)
    case 'Union': {
      const nullable = nullableUnionElement(type)
      if (nullable) {
        const element = goType(program, nullable, options)
        return {
          type: ay.code`Nullable[${element.type}]`,
          text:
            element.text === undefined
              ? undefined
              : `Nullable[${element.text}]`,
          nilable: false,
          jsonNullable: true,
        }
      }

      const oneOrMany = oneOrManyElement(program, type)
      if (oneOrMany) {
        const element = goType(program, oneOrMany, options)
        return {
          type: ay.code`OneOrMany[${element.type}]`,
          text:
            element.text === undefined
              ? undefined
              : `OneOrMany[${element.text}]`,
          nilable: false,
        }
      }

      const name = optionalTypeName(program, type)
      if (!name) {
        const variants = [...type.variants.values()]
          .map((variant) => variant.type)
          .filter((variant) => variant.kind !== 'Intrinsic')
        if (variants.length === 1) {
          return goType(program, variants[0]!, options)
        }
        if (
          variants.length > 1 &&
          variants.every((variant) => variant.kind === 'String')
        ) {
          return { type: 'string', text: 'string', nilable: false }
        }
        throw new Error(
          `anonymous union of [${[...type.variants.values()]
            .map((variant) => variant.type.kind)
            .join(
              ', ',
            )}] is not representable in Go; name the union or reduce it to string literals or a single concrete variant`,
        )
      }
      const runtime = runtimeFilterTypeName(name)
      if (runtime !== name) {
        return { type: runtime, text: runtime, nilable: false }
      }
      const reference = goReferenceTypeName(program, type, name, options.mode)
      return { type: reference, text: reference, nilable: false }
    }
    case 'Enum': {
      const name = typeName(program, type)
      return { type: name, text: name, nilable: false }
    }
    case 'EnumMember': {
      const name = typeName(program, type.enum)
      return { type: name, text: name, nilable: false }
    }
    case 'Model': {
      const typekit = $(program)
      if (typekit.array.is(type)) {
        const element = type.indexer?.value
        if (!element) {
          throw new Error('array model is missing its element type')
        }
        const mapped = goType(program, element, options)
        return {
          type: ay.code`[]${mapped.type}`,
          text: mapped.text === undefined ? undefined : `[]${mapped.text}`,
          nilable: true,
        }
      }
      if (typekit.record.is(type)) {
        const value = type.indexer?.value
        if (!value) {
          throw new Error('record model is missing its value type')
        }
        const mapped = goType(program, value, options)
        return {
          type: ay.code`map[string]${mapped.type}`,
          text:
            mapped.text === undefined ? undefined : `map[string]${mapped.text}`,
          nilable: true,
        }
      }

      const name = optionalTypeName(program, type)
      if (name) {
        const reference = goReferenceTypeName(program, type, name, options.mode)
        return { type: reference, text: reference, nilable: false }
      }

      return {
        type: (
          <go.StructDeclaration>
            <ay.List hardline>
              {goFields(program, type, options).map((field) => (
                <go.StructMember
                  name={field.name}
                  type={field.type}
                  doc={field.doc}
                  tag={{
                    json: `${field.wireName}${field.optional ? ',omitempty' : ''}`,
                  }}
                />
              ))}
            </ay.List>
          </go.StructDeclaration>
        ),
        nilable: false,
      }
    }
    case 'Intrinsic':
      if (type.name === 'unknown') {
        return { type: 'any', text: 'any', nilable: false }
      }
      throw new Error(
        `intrinsic type ${type.name} is not representable in Go; only an explicit unknown maps to any`,
      )
    default:
      throw new Error(`unsupported TypeSpec type kind ${type.kind}`)
  }
}

// Named *FieldFilter unions are runtime-backed (see runtime-symbols.ts):
// GoModels never declares them, so every reference must resolve to one of the
// static filter types shipped in the runtime templates.
const runtimeFilterTypesByUnionName = new Map<string, string>([
  ['StringFieldFilter', 'StringFilter'],
  ['StringFieldFilterExact', 'StringExactFilter'],
  ['ULIDFieldFilter', 'StringExactFilter'],
  ['DateTimeFieldFilter', 'DateTimeFilter'],
  ['NumericFieldFilter', 'NumericFilter'],
  ['BooleanFieldFilter', 'BooleanFilter'],
])

export function runtimeFilterTypeName(name: string): string {
  const runtime = runtimeFilterTypesByUnionName.get(name)
  if (runtime) {
    return runtime
  }

  // Same suffix classification runtime-symbols.ts uses to skip declaring
  // these unions; an unmapped one would reference an undeclared Go type.
  if (name.endsWith('FieldFilter') || name.endsWith('FieldFilterExact')) {
    throw new Error(
      `field filter union ${name} has no runtime filter type; map it in runtimeFilterTypesByUnionName and back it with a static runtime filter`,
    )
  }

  return name
}

export function nullableUnionElement(type: Type): Type | undefined {
  if (type.kind !== 'Union') {
    return undefined
  }

  const variants = [...type.variants.values()].map((variant) => variant.type)
  const hasNull = variants.some(
    (variant) => variant.kind === 'Intrinsic' && variant.name === 'null',
  )
  if (!hasNull) {
    return undefined
  }

  const concrete = variants.filter(
    (variant) => !(variant.kind === 'Intrinsic' && variant.name === 'null'),
  )
  return concrete.length === 1 ? concrete[0] : undefined
}

export function typeName(program: Program, type: Type): string {
  const name = optionalTypeName(program, type)
  if (!name) {
    throw new Error(
      `anonymous ${type.kind} cannot be emitted as a named Go type`,
    )
  }

  return name
}

export function optionalTypeName(
  program: Program,
  type: Type,
): string | undefined {
  const synthetic = syntheticTypeNames.get(program)?.get(type)
  if (synthetic) {
    return synthetic
  }

  const name = baseGoTypeName(program, type)
  if (!name) {
    return undefined
  }

  const mapped =
    resolvedTypeNames.get(program)?.get(name) ??
    stripOnePrefix(name, stripNamePrefixes.get(program) ?? [])

  switch (mapped) {
    case 'PagePaginatedMeta':
      return 'PaginatedMeta'
    default:
      return mapped
  }
}

function baseGoTypeName(program: Program, type: Type): string | undefined {
  const friendlyName = getFriendlyName(program, type)
  const declaredName =
    'name' in type && typeof type.name === 'string' ? type.name : undefined
  const name = friendlyName ?? declaredName
  return name ? goExportedName(name) : undefined
}

function stripOnePrefix(name: string, prefixes: readonly string[]): string {
  for (const prefix of prefixes) {
    if (
      prefix &&
      name.length > prefix.length &&
      name.startsWith(prefix) &&
      /[A-Z]/.test(name[prefix.length]!)
    ) {
      return name.slice(prefix.length)
    }
  }
  return name
}

function resolveStrippedNames(
  names: Iterable<string>,
  prefixes: readonly string[],
): Map<string, string> {
  const all = [...names]
  const resolved = new Map<string, string>()

  if (prefixes.length === 0) {
    for (const name of all) {
      resolved.set(name, name)
    }
    return resolved
  }

  const originals = new Set(all)
  const candidate = new Map<string, string>()
  for (const name of all) {
    candidate.set(name, stripOnePrefix(name, prefixes))
  }

  const candidateCounts = new Map<string, number>()
  for (const target of candidate.values()) {
    candidateCounts.set(target, (candidateCounts.get(target) ?? 0) + 1)
  }

  for (const name of all) {
    const target = candidate.get(name)!
    const collides =
      target !== name &&
      (originals.has(target) || (candidateCounts.get(target) ?? 0) > 1)
    resolved.set(name, collides ? name : target)
  }

  return resolved
}

export function goExportedName(name: string): string {
  const initialisms = new Map([
    ['api', 'API'],
    ['csv', 'CSV'],
    ['http', 'HTTP'],
    ['id', 'ID'],
    ['json', 'JSON'],
    ['llm', 'LLM'],
    ['sql', 'SQL'],
    ['ulid', 'ULID'],
    ['url', 'URL'],
    ['uuid', 'UUID'],
  ])

  return name
    .replace(/([A-Z]+)([A-Z][a-z])/g, '$1 $2')
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .split(/[\s\-_./]+/)
    .filter(Boolean)
    .map((part) => {
      const initialism = initialisms.get(part.toLowerCase())
      return (
        initialism ?? part.charAt(0).toUpperCase() + part.slice(1).toLowerCase()
      )
    })
    .join('')
}

export function queryScalarKind(
  program: Program,
  type: Type,
): GoQueryScalarKind {
  switch (type.kind) {
    case 'String':
      return 'string'
    case 'Boolean':
      return 'boolean'
    case 'Number':
      return Number.isInteger(type.value) ? 'integer' : 'float'
    case 'Enum': {
      const numeric = [...type.members.values()].some(
        (member) => typeof member.value === 'number',
      )
      return numeric ? 'integer' : 'string'
    }
    case 'EnumMember':
      return queryScalarKind(program, type.enum)
    case 'Union': {
      const concrete = [...type.variants.values()]
        .map((variant) => variant.type)
        .filter((variant) => variant.kind !== 'Intrinsic')
      if (concrete.length === 0) {
        throw new Error('query union has no concrete variants')
      }
      const kinds = new Set(
        concrete.map((variant) => queryScalarKind(program, variant)),
      )
      if (kinds.size !== 1) {
        throw new Error(
          `query union ${optionalTypeName(program, type) ?? '<anonymous>'} mixes incompatible scalar kinds`,
        )
      }
      return [...kinds][0]!
    }
    case 'Scalar':
      return scalarQueryKind(type)
    default:
      throw new Error(
        `unsupported ${type.kind} query scalar ${optionalTypeName(program, type) ?? '<anonymous>'}`,
      )
  }
}

export function queryFilterKind(
  program: Program,
  type: Type,
): GoQueryFilterKind {
  switch (optionalTypeName(program, type)) {
    case 'StringFieldFilter':
      return 'string'
    case 'StringFieldFilterExact':
    case 'ULIDFieldFilter':
      return 'stringExact'
    case 'DateTimeFieldFilter':
      return 'dateTime'
    case 'NumericFieldFilter':
      return 'numeric'
    case 'BooleanFieldFilter':
      return 'boolean'
    case 'LabelsFieldFilter':
      return 'labels'
    default:
      queryScalarKind(program, type)
      return 'scalar'
  }
}

function scalarType(scalar: Scalar): GoTypeResult {
  for (
    let current: Scalar | undefined = scalar;
    current;
    current = current.baseScalar
  ) {
    switch (current.name) {
      case 'DateTime':
      case 'utcDateTime':
      case 'offsetDateTime':
        return { type: go.std.time.Time, text: 'time.Time', nilable: false }
      case 'Numeric':
        return { type: 'Numeric', text: 'Numeric', nilable: false }
      case 'boolean':
        return { type: 'bool', text: 'bool', nilable: false }
      case 'integer':
      case 'safeint':
        // `integer` is arbitrary-precision and `safeint` spans 53 bits on the
        // wire, so neither fits a narrower sized Go integer by declaration.
        return { type: 'int64', text: 'int64', nilable: false }
      case 'int8':
        return { type: 'int8', text: 'int8', nilable: false }
      case 'int16':
        return { type: 'int16', text: 'int16', nilable: false }
      case 'int32':
        return { type: 'int32', text: 'int32', nilable: false }
      case 'int64':
        return { type: 'int64', text: 'int64', nilable: false }
      case 'uint8':
        return { type: 'uint8', text: 'uint8', nilable: false }
      case 'uint16':
        return { type: 'uint16', text: 'uint16', nilable: false }
      case 'uint32':
        return { type: 'uint32', text: 'uint32', nilable: false }
      case 'uint64':
        return { type: 'uint64', text: 'uint64', nilable: false }
      case 'float':
      case 'float64':
        return { type: 'float64', text: 'float64', nilable: false }
      case 'float32':
        return { type: 'float32', text: 'float32', nilable: false }
      case 'decimal':
      case 'decimal128':
        return { type: 'Numeric', text: 'Numeric', nilable: false }
      case 'string':
        return { type: 'string', text: 'string', nilable: false }
      default:
        break
    }
  }

  return { type: 'string', text: 'string', nilable: false }
}

function scalarQueryKind(scalar: Scalar): GoQueryScalarKind {
  for (
    let current: Scalar | undefined = scalar;
    current;
    current = current.baseScalar
  ) {
    switch (current.name) {
      case 'DateTime':
      case 'utcDateTime':
      case 'offsetDateTime':
        return 'dateTime'
      case 'boolean':
        return 'boolean'
      case 'integer':
      case 'int8':
      case 'int16':
      case 'int32':
      case 'int64':
      case 'safeint':
      case 'uint8':
      case 'uint16':
      case 'uint32':
      case 'uint64':
        return 'integer'
      case 'float':
      case 'float32':
      case 'float64':
      case 'decimal':
      case 'decimal128':
        return 'float'
      case 'string':
      case 'Numeric':
        return 'string'
      default:
        break
    }
  }

  return 'string'
}

export function oneOrManyElement(
  program: Program,
  type: Type,
): Type | undefined {
  if (type.kind !== 'Union') {
    return undefined
  }

  const variants = [...type.variants.values()]
    .map((variant) => variant.type)
    .filter((variant) => variant.kind !== 'Intrinsic')
  if (variants.length !== 2) {
    return undefined
  }

  const typekit = $(program)
  const array = variants.find(
    (variant): variant is Model =>
      variant.kind === 'Model' && typekit.array.is(variant),
  )
  const single = variants.find((variant) => variant !== array)
  const element = array?.indexer?.value
  if (!array || !single || !element) {
    return undefined
  }

  return element === single ? single : undefined
}
