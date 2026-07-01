import {
  type Model,
  type Namespace,
  type Operation,
  type Program,
  resolveEncodedName,
  type Type,
  type Union,
} from '@typespec/compiler'
import { $ } from '@typespec/compiler/typekit'
import '@typespec/http/experimental/typekit'
import { isCasingDerivable } from './casing.js'
import { bodyProperties } from './utils.jsx'

/**
 * The JSON wire name of a body property: its `@encodedName("application/json", …)`
 * when present, otherwise its declared name. This is the same source the OpenAPI
 * emitter uses, so the gate measures the public→snake transform against the real
 * wire contract rather than against the (camelized) emitted key.
 */
function wireName(program: Program, prop: Type & { name: string }): string {
  return resolveEncodedName(program, prop, 'application/json')
}

/**
 * Fails the build when an emitted wire key is not recoverable from its public
 * (camelized) form by the deterministic casing rule. The boundary mapper derives
 * every wire key it does not carry an explicit name for via `toSnakeCase`, so a
 * non-derivable name would silently ship a wrong key; this gate turns that into a
 * codegen error. Covers body property names, query parameter names, and the
 * discriminator/envelope keys of discriminated unions — every key the mapper or
 * the URL serializer rewrites.
 */
export function assertCasingDerivable(
  program: Program,
  models: Model[],
  operations: Operation[],
): void {
  const tk = $(program)
  const violations: string[] = []
  const check = (where: string, name: string): void => {
    if (!isCasingDerivable(name)) {
      violations.push(`${where}: '${name}' is not snake↔camel derivable`)
    }
  }

  for (const model of models) {
    for (const prop of bodyProperties(program, model)) {
      check(
        `${model.name}.${prop.name}`,
        wireName(program, prop as Type & { name: string }),
      )
    }
  }

  for (const op of operations) {
    const httpOp = tk.httpOperation.get(op)
    for (const param of httpOp.parameters.parameters) {
      if (param.type === 'query') {
        check(`${op.name} query`, param.name)
      }
    }
  }

  // Unions the boundary mapper actually walks: those reachable from a request body
  // or a success response. Error-envelope unions (e.g. `InvalidParameter` via
  // `badRequest`) are excluded — they are consumed by `to-error.ts`, never mapped.
  const mappedUnions = mappedReachableUnions(program, operations)
  const ambiguousUnions: string[] = []
  for (const union of userUnions(program)) {
    const discriminated = tk.union.getDiscriminatedUnion(union)
    if (!discriminated) {
      // A non-discriminated union of two or more object variants has no key the
      // mapper can use to pick a variant; it would have to guess from the data's
      // key set at runtime. The mapper deliberately does not — so fail the build,
      // forcing the union to be `@discriminated` (scalar-vs-object unions are fine,
      // the mapper distinguishes those by JS type).
      if (mappedUnions.has(union) && objectVariantCount(program, union) >= 2) {
        ambiguousUnions.push(union.name ?? '<anonymous union>')
      }
      continue
    }
    check(
      `${union.name ?? 'union'} discriminator`,
      discriminated.options.discriminatorPropertyName,
    )
    if (discriminated.options.envelope === 'object') {
      check(
        `${union.name ?? 'union'} envelope`,
        discriminated.options.envelopePropertyName,
      )
    }
  }

  if (ambiguousUnions.length > 0) {
    throw new Error(
      `camelCase SDK: ${ambiguousUnions.length} non-discriminated union(s) with ` +
        `multiple object variants cannot be mapped (the wire mapper cannot pick a ` +
        `variant). Add @discriminated.\n  ${ambiguousUnions.join('\n  ')}`,
    )
  }

  if (violations.length > 0) {
    throw new Error(
      `camelCase SDK: ${violations.length} wire key(s) are not casing-derivable. ` +
        `Add an @encodedName or an explicit override.\n  ${violations.join('\n  ')}`,
    )
  }
}

/** The number of a union's variants whose value is an object/model type. */
function objectVariantCount(program: Program, union: Union): number {
  const tk = $(program)
  let count = 0
  for (const variant of union.variants.values()) {
    const type = variant.type
    if (type.kind === 'Model' && !tk.array.is(type) && !tk.record.is(type)) {
      count++
    }
  }
  return count
}

/**
 * The unions the boundary mapper walks: those in the transitive closure of every
 * operation's request body and success-response body. Error responses are excluded
 * (their bodies are read by the error path, not mapped).
 */
function mappedReachableUnions(
  program: Program,
  operations: Operation[],
): Set<Union> {
  const tk = $(program)
  const unions = new Set<Union>()
  const seen = new Set<Type>()
  const visit = (type: Type | undefined): void => {
    if (!type || seen.has(type)) {
      return
    }
    seen.add(type)
    switch (type.kind) {
      case 'Union':
        unions.add(type)
        for (const variant of type.variants.values()) {
          visit(variant.type)
        }
        break
      case 'Model':
        if (type.indexer) {
          visit(type.indexer.value)
        }
        if (type.baseModel) {
          visit(type.baseModel)
        }
        for (const prop of type.properties.values()) {
          visit(prop.type)
        }
        break
      case 'Tuple':
        for (const value of type.values) {
          visit(value)
        }
        break
      default:
        break
    }
  }

  for (const op of operations) {
    const httpOp = tk.httpOperation.get(op)
    visit(httpOp.parameters.body?.type)
    for (const response of httpOp.responses) {
      if (!isSuccessStatus(response.statusCodes)) {
        continue
      }
      for (const content of response.responses) {
        visit(content.body?.type)
      }
    }
  }
  return unions
}

function isSuccessStatus(
  statusCodes: number | '*' | { start: number; end: number },
): boolean {
  if (statusCodes === '*') {
    return false
  }
  if (typeof statusCodes === 'number') {
    return statusCodes >= 200 && statusCodes < 300
  }
  return statusCodes.start >= 200 && statusCodes.start < 300
}

function userUnions(program: Program): Union[] {
  const tk = $(program)
  const globalNs = program.getGlobalNamespaceType()
  const result: Union[] = []
  const walk = (ns: Namespace): void => {
    if (ns !== globalNs && !tk.type.isUserDefined(ns)) {
      return
    }
    result.push(...ns.unions.values())
    ns.namespaces.forEach(walk)
  }
  walk(globalNs)
  return result
}
