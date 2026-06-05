import { type Model, type Program, type Type } from '@typespec/compiler'
import { $ } from '@typespec/compiler/typekit'
import { isHttpEnvelopeProperty } from './utils.jsx'
import { type IoMode, type RefName, isOptional, tsTypeOf } from './ts-types.js'
import { computeDivergentModels, inputVariantName } from './input-variants.js'

type ResolveName = (type: Type) => string | undefined
type SchemaName = (resolvedName: string) => string

function jsdoc(doc: string | undefined, indent: string): string | undefined {
  if (!doc) {
    return undefined
  }
  const text = doc.trim().replace(/\s+/g, ' ')
  return `${indent}/** ${text} */`
}

/**
 * A mutual-assignability check between an emitted type and its wire schema. The
 * emitted type is walked from TypeSpec independently of zod, so this is the only
 * place the two artifacts meet: any divergence becomes a build error.
 */
function conformanceGuard(name: string, schemaRef: string): string {
  return (
    `type _Assert${name} = [${name}] extends [${schemaRef}]\n` +
    `  ? [${schemaRef}] extends [${name}]\n` +
    `    ? true\n` +
    `    : { __error: '${name} is missing fields present in the wire schema' }\n` +
    `  : { __error: '${name} has fields not present in the wire schema' }\n` +
    `const _assert${name}: _Assert${name} = true`
  )
}

/**
 * A one-directional check for input variants: the emitted type must be a valid
 * input the schema accepts (`[X] extends [z.input]`), but not vice versa. zod's
 * `z.input` of a coerced leaf (e.g. `z.coerce.bigint()`) is the loose `unknown`;
 * the emitted type deliberately keeps the strict leaf (`bigint`), so the reverse
 * direction is intentionally dropped instead of widening the public type.
 */
function inputConformanceGuard(name: string, schemaRef: string): string {
  return (
    `type _Assert${name} = [${name}] extends [${schemaRef}]\n` +
    `  ? true\n` +
    `  : { __error: '${name} is not assignable to the wire input schema' }\n` +
    `const _assert${name}: _Assert${name} = true`
  )
}

function interfaceBody(
  program: Program,
  model: Model,
  refName: RefName,
  io: IoMode,
): string {
  const tk = $(program)
  const lines: string[] = []
  for (const prop of model.properties.values()) {
    if (isHttpEnvelopeProperty(program, prop)) {
      continue
    }
    const doc = jsdoc(tk.type.getDoc(prop), '  ')
    if (doc) {
      lines.push(doc)
    }
    const opt = isOptional(prop, io) ? '?' : ''
    lines.push(
      `  ${prop.name}${opt}: ${tsTypeOf(program, prop.type, refName, io)}`,
    )
  }
  // An indexer (`...Record<...>`) makes the model open; mirror it with an index
  // signature so the interface stays assignable to the open wire shape while
  // keeping its documented known fields.
  if (model.indexer?.key.name === 'string') {
    lines.push(
      `  [key: string]: ${tsTypeOf(program, model.indexer.value, refName, io)}`,
    )
  }
  return lines.join('\n')
}

export interface InterfacesResult {
  types: string
  asserts: string
  /** Names of every interface/type exported from `types.ts`, including `…Input` variants. */
  typeNames: string[]
  /** Models whose input shape diverges from their output interface. */
  divergentModels: Set<Model>
  /**
   * Input-mode ref resolver: the `…Input` variant for a divergent model, else
   * the model's interface. Used to build request/query types.
   */
  refNameInput: RefName
}

export function interfacesFile(
  program: Program,
  models: Model[],
  resolveName: ResolveName,
  schemaName: SchemaName,
  interfaceName: (resolvedName: string) => string,
): InterfacesResult {
  const tk = $(program)
  const emitted = new Set<Type>(models)
  const refName = (type: Type): string | undefined => {
    if (!emitted.has(type)) {
      return undefined
    }
    const resolved = resolveName(type)
    return resolved ? interfaceName(resolved) : undefined
  }

  const divergent = computeDivergentModels(program, models)
  // An input-mode ref points at a child's `…Input` variant when that child also
  // diverges, so relaxed optionality propagates through the subtree.
  const refNameInput: RefName = (type) => {
    const name = refName(type)
    if (!name) {
      return undefined
    }
    return type.kind === 'Model' && divergent.has(type)
      ? inputVariantName(name)
      : name
  }

  const blocks: string[] = []
  const inputBlocks: string[] = []
  const asserts: string[] = []
  for (const model of models) {
    const resolved = resolveName(model)
    if (!resolved) {
      continue
    }
    const name = interfaceName(resolved)
    const schemaRef = `z.output<typeof schemas.${schemaName(resolved)}>`
    const doc = jsdoc(tk.type.getDoc(model), '')

    // When the model `extends` an emitted base, the interface extends it too so
    // inherited fields (and their docs) propagate; only own properties go in the
    // body.
    const baseName = model.baseModel ? refName(model.baseModel) : undefined
    const extendsClause = baseName ? ` extends ${baseName}` : ''

    const hasWireProps = [...model.properties.values()].some(
      (prop) => !isHttpEnvelopeProperty(program, prop),
    )

    // Models with no wire-mapped properties and no base (records, unions, marker
    // types) have no structural interface — alias straight to the mapped type so
    // it is correct (e.g. `Labels` -> `Record<string, string>`) rather than an
    // empty, permissive `interface {}`.
    if (!hasWireProps && !baseName) {
      const parts: string[] = []
      if (doc) {
        parts.push(doc)
      }
      // Exclude the model from its own ref resolution so the alias expresses its
      // structure (`Record<string, string>`) instead of aliasing to itself.
      const structural: RefName = (type) =>
        type === model ? undefined : refName(type)
      parts.push(
        `export type ${name} = ${tsTypeOf(program, model, structural)}`,
      )
      blocks.push(parts.join('\n'))
      // The alias is an independently-walked type, not the inferred shape, so it
      // is guarded too — unlike a `z.output` alias, it can diverge.
      asserts.push(conformanceGuard(name, schemaRef))
      continue
    }

    const body = interfaceBody(program, model, refName, 'output')
    const parts: string[] = []
    if (doc) {
      parts.push(doc)
    }
    parts.push(`export interface ${name}${extendsClause} {\n${body}\n}`)
    blocks.push(parts.join('\n'))
    asserts.push(conformanceGuard(name, schemaRef))

    // The input variant relaxes defaulted fields to optional (transitively); it
    // is emitted only where the input shape actually diverges from the output.
    if (divergent.has(model)) {
      const inputNameStr = inputVariantName(name)
      const inputExtends =
        model.baseModel && divergent.has(model.baseModel)
          ? ` extends ${inputVariantName(refName(model.baseModel)!)}`
          : extendsClause
      const inputBody = interfaceBody(program, model, refNameInput, 'input')
      inputBlocks.push(
        `export interface ${inputNameStr}${inputExtends} {\n${inputBody}\n}`,
      )
      asserts.push(
        inputConformanceGuard(
          inputNameStr,
          `z.input<typeof schemas.${schemaName(resolved)}>`,
        ),
      )
    }
  }

  const allBlocks = [...blocks, ...inputBlocks]
  const typeNames = allBlocks
    .map((b) => b.match(/export (?:interface|type) (\w+)/)?.[1])
    .filter((n): n is string => Boolean(n))
  const types = `${allBlocks.join('\n\n')}\n`
  const assertImports =
    `import { z } from 'zod'\n` +
    `import * as schemas from './schemas.js'\n` +
    `import type { ${typeNames.join(', ')} } from './types.js'\n`
  const assertsFile = `${assertImports}\n${asserts.join('\n\n')}\n`
  return {
    types,
    asserts: assertsFile,
    typeNames,
    divergentModels: divergent,
    refNameInput,
  }
}
