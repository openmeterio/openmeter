import {
  type Model,
  type Operation,
  type Program,
  resolveEncodedName,
  type Type,
  type Union,
} from '@typespec/compiler'
import { $ } from '@typespec/compiler/typekit'
import '@typespec/http/experimental/typekit'
import { isCasingDerivable, toCamelCase } from './casing.js'
import { bodyProperties, emitsAsIntersection } from './utils.jsx'

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

  // Two distinct wire names in the same object can camelize to the same public
  // key (e.g. `foo_bar` and `fooBar`); each individually round-trips fine, so
  // `check` above never catches it, but the emitted model/query type would
  // silently duplicate or shadow one side. Fail the build instead.
  const collisions: string[] = []
  const checkNoCollisions = (where: string, wireNames: string[]): void => {
    const byCamel = new Map<string, Set<string>>()
    for (const name of wireNames) {
      const camel = toCamelCase(name)
      const group = byCamel.get(camel) ?? new Set<string>()
      group.add(name)
      byCamel.set(camel, group)
    }
    for (const [camel, names] of byCamel) {
      if (names.size > 1) {
        collisions.push(
          `${where}: ${[...names].join(', ')} all camelize to '${camel}'`,
        )
      }
    }
  }

  for (const model of models) {
    const wireNames: string[] = []
    for (const prop of bodyProperties(program, model)) {
      const name = wireName(program, prop as Type & { name: string })
      check(`${model.name}.${prop.name}`, name)
      wireNames.push(name)
    }
    checkNoCollisions(model.name || '<anonymous model>', wireNames)
  }

  for (const op of operations) {
    const httpOp = tk.httpOperation.get(op)
    const queryNames: string[] = []
    for (const param of httpOp.parameters.parameters) {
      if (param.type === 'query') {
        check(`${op.name} query`, param.name)
        queryNames.push(param.name)
      }
    }
    checkNoCollisions(`${op.name} query`, queryNames)
  }

  if (collisions.length > 0) {
    throw new Error(
      `camelCase SDK: ${collisions.length} wire key group(s) collapse onto the ` +
        `same camelCase name. Rename one side or add an explicit override.\n  ${collisions.join('\n  ')}`,
    )
  }

  // Unions the boundary mapper actually walks: those reachable from a request body
  // or a success response. Error-envelope unions (e.g. `InvalidParameter` via
  // `badRequest`) are excluded — they are consumed by `to-error.ts`, never mapped.
  const { unions: mappedUnions, models: mappedModels } = mappedReachableTypes(
    program,
    operations,
  )

  // Models the mapper would walk that emit as `z.intersection(...)` (a record
  // spread combined with named fields): the walker's object/record branches
  // dispatch on `def.type`, and zod has no `"intersection"` case in that
  // dispatch, so such a schema would silently pass through untransformed —
  // the record side keeps its wire casing, the named-field side keeps its
  // wire casing too, and nothing gets camelized. Every intersection model
  // today (`baseError`/`badRequest`) is error-envelope-only and bypasses the
  // mapper entirely, so this fails the build the moment that stops being
  // true rather than letting a future model silently ship the wrong casing.
  const intersectionModels: string[] = []
  for (const model of mappedModels) {
    if (emitsAsIntersection(program, model)) {
      intersectionModels.push(model.name || '<anonymous model>')
    }
  }
  if (intersectionModels.length > 0) {
    throw new Error(
      `camelCase SDK: ${intersectionModels.length} model(s) reachable from a ` +
        `request body or success response emit as z.intersection(...), which the ` +
        `wire mapper cannot walk (no case for it). Restructure the model to avoid ` +
        `combining a record indexer with named properties, or extend the mapper's ` +
        `walk() with an intersection branch.\n  ${intersectionModels.join('\n  ')}`,
    )
  }

  const ambiguousUnions: string[] = []
  // Iterates `mappedUnions` (every union reachable from an operation's request/
  // response body, named or inline) rather than only namespace-owned unions, so
  // an inline union in a model field still reaches the ambiguous-union and
  // discriminator/envelope casing checks below.
  for (const union of mappedUnions) {
    const discriminated = tk.union.getDiscriminatedUnion(union)
    if (!discriminated) {
      // A non-discriminated union of two or more object variants has no key the
      // mapper can use to pick a variant; it would have to guess from the data's
      // key set at runtime. The mapper deliberately does not — so fail the build,
      // forcing the union to be `@discriminated` (scalar-vs-object unions are fine,
      // the mapper distinguishes those by JS type).
      if (objectVariantCount(program, union) >= 2) {
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
 * The unions and models the boundary mapper walks: those in the transitive closure
 * of every operation's request body and success-response body. Error responses are
 * excluded (their bodies are read by the error path, not mapped).
 */
function mappedReachableTypes(
  program: Program,
  operations: Operation[],
): { unions: Set<Union>; models: Set<Model> } {
  const tk = $(program)
  const unions = new Set<Union>()
  const models = new Set<Model>()
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
        models.add(type)
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
  return { unions, models }
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
