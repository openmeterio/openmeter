import {
  type ModelProperty,
  type Operation,
  type Program,
  type Type,
} from '@typespec/compiler'
import { $ } from '@typespec/compiler/typekit'
import { operationBaseName } from './ZodOperations.jsx'
import { type RefName, isOptional, tsTypeOf } from './ts-types.js'
import type { SdkOperation } from './sdk-operations.js'

interface QueryLeaf {
  name: string
  prop: ModelProperty
}

function jsdoc(doc: string | undefined): string | undefined {
  if (!doc) {
    return undefined
  }
  return `  /** ${doc.trim().replace(/\s+/g, ' ')} */`
}

function queryLeaves(program: Program, op: Operation): QueryLeaf[] {
  const httpOp = $(program).httpOperation.get(op)
  const leaves: QueryLeaf[] = []
  for (const param of httpOp.parameters.parameters) {
    if (param.type === 'query') {
      leaves.push({ name: param.name, prop: param.param })
    }
  }
  return leaves
}

/**
 * The per-operation query type, walked from its parameter leaves in input mode.
 * Query params arrive as strings on the wire; their coerced leaves keep the
 * strict mapped type (the one-directional guard certifies they are valid
 * inputs).
 */
function queryType(
  program: Program,
  op: Operation,
  base: string,
  refNameInput: RefName,
): string {
  const tk = $(program)
  const lines: string[] = []
  for (const { name, prop } of queryLeaves(program, op)) {
    const doc = jsdoc(tk.type.getDoc(prop))
    if (doc) {
      lines.push(doc)
    }
    const opt = isOptional(prop, 'input') ? '?' : ''
    lines.push(
      `  ${name}${opt}: ${tsTypeOf(program, prop.type, refNameInput, 'input')}`,
    )
  }
  return `export interface ${base}Query {\n${lines.join('\n')}\n}`
}

function inputGuard(name: string, schemaRef: string): string {
  return (
    `type _Assert${name} = [${name}] extends [${schemaRef}]\n` +
    `  ? true\n` +
    `  : { __error: '${name} is not assignable to the wire input schema' }\n` +
    `const _assert${name}: _Assert${name} = true`
  )
}

function requestDecl(op: SdkOperation, bodyRef: string): string {
  const hasPath = op.pathParams.length > 0
  const hasQuery = op.queryParams.length > 0
  const decl = `export type ${op.base}Request =`
  const pathField = (p: string) => `  ${p}: string`
  const pathObj = `{ ${op.pathParams.map((p) => `${p}: string`).join('; ')} }`
  const queryRef = `${op.base}Query`

  if (op.hasBody) {
    if (!hasPath && !hasQuery) {
      return `${decl} ${bodyRef}`
    }
    const queryPart = hasQuery ? ` & ${queryRef}` : ''
    const pathLine = hasPath
      ? `${op.pathParams.map(pathField).join('\n')}\n`
      : ''
    return `${decl} {\n${pathLine}  body: ${bodyRef}\n}${queryPart}`
  }
  if (hasQuery) {
    return hasPath ? `${decl} ${queryRef} & ${pathObj}` : `${decl} ${queryRef}`
  }
  if (hasPath) {
    return `${decl} {\n${op.pathParams.map(pathField).join('\n')}\n}`
  }
  return `${decl} Record<string, never>`
}

function responseDecl(op: SdkOperation): string {
  if (!op.hasResponse) {
    return `export type ${op.base}Response = void`
  }
  if (op.responseInterface) {
    return `export type ${op.base}Response = ${op.responseInterface}`
  }
  return `export type ${op.base}Response = z.output<typeof schemas.${op.funcName}Response>`
}

export interface RequestTypes {
  /** The request/response type declarations (request, query, response). */
  decls: string
  /** One-directional input guards for the per-operation query types. */
  guards: string
  /** Interface imports from `../models/types.js`, mapping name to local alias. */
  interfaceImports: Map<string, string>
  /** Whether any declaration references zod (`z`/`schemas`). */
  usesZod: boolean
}

export function requestTypesFor(
  program: Program,
  ops: Operation[],
  sdkOps: SdkOperation[],
  refNameInput: RefName,
  jsonBodyOverrides: Map<string, Type>,
): RequestTypes {
  const opByBase = new Map(
    ops.map((op) => [operationBaseName(program, op), op]),
  )
  // A request type alias `<Base>Request` collides with a body model interface of
  // the same name; such a body is imported under a `<Name>Body` alias so the
  // local request declaration owns the name.
  const localNames = new Set(sdkOps.map((op) => `${op.base}Request`))
  const interfaceImports = new Map<string, string>()
  const importAlias = (name: string): string => {
    if (interfaceImports.has(name)) {
      return interfaceImports.get(name)!
    }
    const alias = localNames.has(name) ? `${name}Body` : name
    interfaceImports.set(name, alias)
    return alias
  }

  const blocks: string[] = []
  const guards: string[] = []
  let usesZod = false

  for (const sdkOp of sdkOps) {
    const jsonBody = jsonBodyOverrides.get(sdkOp.base)
    let bodyRef: string
    if (jsonBody) {
      bodyRef = tsTypeOf(program, jsonBody, refNameInput, 'input')
      for (const name of collectTypeRefs(jsonBody, refNameInput)) {
        importAlias(name)
      }
    } else if (sdkOp.requestBodyInterface) {
      bodyRef = importAlias(sdkOp.requestBodyInterface)
    } else {
      bodyRef = 'Record<string, never>'
    }
    if (sdkOp.responseInterface) {
      importAlias(sdkOp.responseInterface)
    }

    if (sdkOp.queryParams.length > 0) {
      const op = opByBase.get(sdkOp.base)!
      blocks.push(queryType(program, op, sdkOp.base, refNameInput))
      guards.push(
        inputGuard(
          `${sdkOp.base}Query`,
          `z.input<typeof schemas.${sdkOp.funcName}QueryParams>`,
        ),
      )
      usesZod = true
      for (const name of collectQueryInterfaceImports(
        program,
        op,
        refNameInput,
      )) {
        importAlias(name)
      }
    }

    const response = responseDecl(sdkOp)
    if (response.includes('z.output')) {
      usesZod = true
    }
    blocks.push(`${requestDecl(sdkOp, bodyRef)}\n${response}`)
  }

  return {
    decls: blocks.join('\n\n'),
    guards: guards.join('\n\n'),
    interfaceImports,
    usesZod,
  }
}

/** The named interfaces a type references, so funcs can import them. Stops at
 * each named ref (which becomes an import) rather than descending into it. */
function collectTypeRefs(root: Type, refNameInput: RefName): Set<string> {
  const names = new Set<string>()
  const visit = (type: Type): void => {
    const name = refNameInput(type)
    if (name) {
      names.add(name)
      return
    }
    if (type.kind === 'Model') {
      if (type.indexer) {
        visit(type.indexer.value)
      }
      for (const prop of type.properties.values()) {
        visit(prop.type)
      }
    } else if (type.kind === 'Union') {
      for (const variant of type.variants.values()) {
        visit(variant.type)
      }
    } else if (type.kind === 'ModelProperty') {
      visit(type.type)
    }
  }
  visit(root)
  return names
}

/** The named interfaces a query type references, so funcs can import them. */
function collectQueryInterfaceImports(
  program: Program,
  op: Operation,
  refNameInput: RefName,
): Set<string> {
  const names = new Set<string>()
  for (const { prop } of queryLeaves(program, op)) {
    for (const name of collectTypeRefs(prop.type, refNameInput)) {
      names.add(name)
    }
  }
  return names
}
