import type { SdkOperation } from './sdk-operations.js'
import { namespaceNames } from './sdk-operations.js'
import type { RequestTypes } from './request-types.js'

function pathExpr(op: SdkOperation): string {
  const template = op.path.replace(/^\//, '')
  const params = op.pathParams.map((p) => `${p}: req.${p}`).join(', ')
  return `encodePath('${template}', { ${params} })`
}

function funcBody(op: SdkOperation): string {
  const reqType = `${op.base}Request`
  const resType = `${op.base}Response`
  const hasPath = op.pathParams.length > 0
  const hasQuery = op.queryParams.length > 0

  const reqParam =
    hasQuery && !op.hasBody && !hasPath
      ? `req: ${reqType} = {}`
      : `req: ${reqType}`

  const url = hasPath ? pathExpr(op) : `'${op.path.replace(/^\//, '')}'`

  const kyOpts: string[] = []
  if (hasQuery) {
    kyOpts.push('searchParams')
  }
  if (op.hasBody) {
    const wrapped = hasPath || hasQuery
    kyOpts.push(wrapped ? 'json: req.body' : 'json: req')
  }
  if (op.textResponseContentType) {
    kyOpts.push('headers')
  }
  const optsObj =
    kyOpts.length > 0 ? `{ ...options, ${kyOpts.join(', ')} }` : 'options'

  const lines: string[] = []
  lines.push(
    `export function ${op.funcName}(`,
    `  client: Client,`,
    `  ${reqParam},`,
    `  options?: RequestOptions,`,
    `): Promise<Result<${resType}>> {`,
  )
  if (hasQuery) {
    const entries = op.queryParams.map((p) =>
      p === 'sort' ? `    sort: encodeSort(req.sort),` : `    ${p}: req.${p},`,
    )
    lines.push(`  const searchParams = toURLSearchParams({`, ...entries, `  })`)
  }
  if (op.textResponseContentType) {
    // The server negotiates this variant on the exact Accept media type; user
    // headers are carried over so only `accept` is forced.
    lines.push(
      `  const headers = new Headers(options?.headers as HeadersInit | undefined)`,
      `  headers.set('accept', '${op.textResponseContentType}')`,
    )
  }
  if (hasPath) {
    lines.push(`  const path = ${url}`)
  }
  const target = hasPath ? 'path' : url
  const optsArg = optsObj === 'options' ? ', options' : `, ${optsObj}`
  if (op.hasResponse) {
    lines.push(
      `  return request(() =>`,
      `    http(client)`,
      `      .${op.verb}(${target}${optsArg})`,
      op.textResponseContentType
        ? `      .text(),`
        : `      .json<${resType}>(),`,
      `  )`,
      `}`,
    )
  } else {
    lines.push(
      `  return request(async () => {`,
      `    await http(client).${op.verb}(${target}${optsArg})`,
      `  })`,
      `}`,
    )
  }
  return lines.join('\n')
}

/**
 * The per-namespace operation types file (`models/operations/<ns>.ts`): the
 * request/response/query type declarations. These are kept out of the public
 * `funcs/<ns>.ts` so that file holds only functions, mirroring the SDK layout.
 */
export function operationsFile(
  tag: string,
  requestTypes: RequestTypes,
): string {
  const imports: string[] = []
  if (requestTypes.usesZod) {
    imports.push(
      `import { z } from 'zod'`,
      `import * as schemas from '../schemas.js'`,
    )
  }
  const interfaceImports = [...requestTypes.interfaceImports.entries()]
    .sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0))
    .map(([name, alias]) => (name === alias ? name : `${name} as ${alias}`))
  if (interfaceImports.length > 0) {
    imports.push(
      `import type { ${interfaceImports.join(', ')} } from '../types.js'`,
    )
  }
  const head = imports.length > 0 ? `${imports.join('\n')}\n\n` : ''
  return `${head}${requestTypes.decls}\n`
}

/**
 * One-directional input guards for a namespace's `…Query` types, in a sibling
 * `models/operations/<ns>.assert.ts` so the types file stays free of value-level
 * guard declarations (matching how model guards live in `types.assert.ts`).
 */
export function operationsAssertFile(
  tag: string,
  requestTypes: RequestTypes,
): string | undefined {
  if (!requestTypes.guards) {
    return undefined
  }
  const file = namespaceFile(tag)
  const queryNames = [
    ...requestTypes.guards.matchAll(/type _Assert(\w+Query) /g),
  ].map((m) => m[1])
  const imports = [
    `import { z } from 'zod'`,
    `import * as schemas from '../schemas.js'`,
    `import type { ${queryNames.join(', ')} } from './${file}.js'`,
  ].join('\n')
  return `${imports}\n\n${requestTypes.guards}\n`
}

export function funcsFile(tag: string, ops: SdkOperation[]): string {
  const file = namespaceFile(tag)
  // The funcs reference only the per-op `…Request`/`…Response` aliases, which
  // live in the operations types module.
  const imports = [
    `import { type Client, http } from '../core.js'`,
    `import { type Result, type RequestOptions } from '../lib/types.js'`,
    `import { request } from '../lib/request.js'`,
    `import { encodePath, toURLSearchParams, encodeSort } from '../lib/encodings.js'`,
    `import type {`,
    ...ops.flatMap((op) => [`  ${op.base}Request,`, `  ${op.base}Response,`]),
    `} from '../models/operations/${file}.js'`,
  ]
  const bodies = ops.map(funcBody).join('\n\n')
  return `${imports.join('\n')}\n\n${bodies}\n`
}

interface FacadeNode {
  className: string
  ops: SdkOperation[]
  children: Map<string, FacadeNode>
}

function emitMethod(op: SdkOperation): string {
  const reqOptional =
    op.queryParams.length > 0 && !op.hasBody && op.pathParams.length === 0
  const reqParam = reqOptional
    ? `request?: ${op.base}Request`
    : `request: ${op.base}Request`
  return [
    `  async ${op.methodName}(`,
    `    ${reqParam},`,
    `    options?: RequestOptions,`,
    `  ): Promise<${op.base}Response> {`,
    `    return unwrap(await ${op.funcName}(this._client, request, options))`,
    `  }`,
  ].join('\n')
}

function emitNode(node: FacadeNode): string[] {
  const getters = [...node.children.entries()].map(([name, child]) => {
    const field = `_${name}`
    return [
      `  private ${field}?: ${child.className}`,
      `  get ${name}(): ${child.className} {`,
      `    return (this.${field} ??= new ${child.className}(this._client))`,
      `  }`,
    ].join('\n')
  })
  const members = [...node.ops.map(emitMethod), ...getters].join('\n\n')

  const klass = [
    `export class ${node.className} {`,
    `  constructor(private readonly _client: Client) {}`,
    ``,
    members,
    `}`,
  ].join('\n')

  const childClasses = [...node.children.values()].flatMap(emitNode)
  return [klass, ...childClasses]
}

export function facadeFile(tag: string, ops: SdkOperation[]): string {
  const { class: cls } = namespaceNames(tag)
  const file = namespaceFile(tag)

  const root: FacadeNode = { className: cls, ops: [], children: new Map() }
  for (const op of ops) {
    let node = root
    for (const segment of op.nestPath) {
      let child = node.children.get(segment)
      if (!child) {
        const childClass = `${node.className}${segment.charAt(0).toUpperCase() + segment.slice(1)}`
        child = { className: childClass, ops: [], children: new Map() }
        node.children.set(segment, child)
      }
      node = child
    }
    node.ops.push(op)
  }

  const classes = emitNode(root).join('\n\n')
  const funcImports = ops.map((op) => `  ${op.funcName},`).join('\n')
  const typeImports = ops
    .flatMap((op) => [`  ${op.base}Request,`, `  ${op.base}Response,`])
    .join('\n')

  return [
    `import { type Client } from '../core.js'`,
    `import { unwrap, type RequestOptions } from '../lib/types.js'`,
    `import {`,
    funcImports,
    `} from '../funcs/${file}.js'`,
    `import type {`,
    typeImports,
    `} from '../models/operations/${file}.js'`,
    ``,
    classes,
    ``,
  ].join('\n')
}

export function sdkRootFile(tags: string[]): string {
  const imports = [
    `import { Client } from '../core.js'`,
    ...tags.map((tag) => {
      const { class: cls } = namespaceNames(tag)
      return `import { ${cls} } from './${namespaceFile(tag)}.js'`
    }),
  ].join('\n')

  const members = tags
    .map((tag) => {
      const { class: cls, getter } = namespaceNames(tag)
      return [
        `  private _${getter}?: ${cls}`,
        `  get ${getter}(): ${cls} {`,
        `    return (this._${getter} ??= new ${cls}(this))`,
        `  }`,
      ].join('\n')
    })
    .join('\n\n')

  return [
    imports,
    ``,
    `export class OpenMeter extends Client {`,
    members,
    `}`,
    ``,
  ].join('\n')
}

export function funcsIndexFile(tags: string[]): string {
  return (
    tags.map((tag) => `export * from './${namespaceFile(tag)}.js'`).join('\n') +
    '\n'
  )
}

export function indexFile(tags: string[], modelTypeNames: string[]): string {
  const namespaceExports = tags
    .map((tag) => {
      const { class: cls } = namespaceNames(tag)
      return `export { ${cls} } from './sdk/${namespaceFile(tag)}.js'`
    })
    .join('\n')
  const operationTypeExports = tags
    .map(
      (tag) =>
        `export type * from './models/operations/${namespaceFile(tag)}.js'`,
    )
    .join('\n')

  // Domain model types are exported by name rather than via `export type *`. Some
  // request models (`CreateMeterRequest`, …) are also re-exported as operation
  // aliases, so a second star would make those names ambiguous (TS2308). An
  // explicit named re-export shadows the operation stars and resolves to the
  // identical underlying type, while making every domain type (`Meter`,
  // `RateCard`, …) importable by name.
  const uniqueTypeNames = [...new Set(modelTypeNames)]
  const modelTypeExports = [
    `export type {`,
    ...uniqueTypeNames.map((name) => `  ${name},`),
    `} from './models/types.js'`,
  ].join('\n')

  return [
    `export { OpenMeter } from './sdk/sdk.js'`,
    namespaceExports,
    `export { Client } from './core.js'`,
    `export { HTTPError } from './models/errors.js'`,
    ``,
    `export { ServerList, Regions } from './lib/config.js'`,
    `export type { SDKOptions, Region, ServerVariables } from './lib/config.js'`,
    `export type { Result, RequestOptions } from './lib/types.js'`,
    ``,
    `export {`,
    `  encodePath,`,
    `  encodeSort,`,
    `  querySerializer,`,
    `  toURLSearchParams,`,
    `} from './lib/encodings.js'`,
    ``,
    `export * as funcs from './funcs/index.js'`,
    ``,
    operationTypeExports,
    ``,
    modelTypeExports,
    ``,
  ].join('\n')
}

export function namespaceFile(tag: string): string {
  return namespaceNames(tag).getter
}
