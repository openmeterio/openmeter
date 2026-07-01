import type { SdkOperation } from './sdk-operations.js'
import { namespaceNames } from './sdk-operations.js'
import type { RequestTypes } from './request-types.js'
import { toCamelCase } from './casing.js'

function pathExpr(op: SdkOperation): string {
  // Inline the path as a template literal: each `{param}` becomes the
  // URL-encoded request field. Path params are typed as required strings, but
  // that only holds under the TS compiler — a caller on `any`/plain JS, or one
  // that builds the request object dynamically, can still omit one. Without a
  // runtime check, `String(undefined)` renders the literal segment "undefined"
  // into the URL, turning a client-side mistake into a confusing server request
  // instead of an immediate, clear error (matching the guard `encodePath` has
  // always thrown for the server-variable baseUrl).
  const path = op.path
    .replace(/^\//, '')
    .replace(
      /\{(\w+)\}/g,
      (_, p: string) =>
        `\${(() => { if (req.${p} === undefined) { throw new Error('missing path parameter: ${p}') } return encodeURIComponent(String(req.${p})) })()}`,
    )
  return `\`${path}\``
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
    // The snake_case wire body is computed into `body` inside the closure (so the
    // optional validate check can run against the actual payload before sending).
    kyOpts.push('json: body')
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
  if (op.textResponseContentType) {
    // The server negotiates this variant on the exact Accept media type; user
    // headers are carried over so only `accept` is forced.
    lines.push(
      `  const headers = new Headers(options?.headers as HeadersInit | undefined)`,
      `  headers.set('accept', '${op.textResponseContentType}')`,
    )
  }
  const target = hasPath ? 'path' : url
  const optsArg = optsObj === 'options' ? ', options' : `, ${optsObj}`
  // The path is built inside the request() closure (not as a top-level `const`
  // above it), so a missing required path param throws from within the wrapped
  // callback and surfaces as Result.error, the same as a body/query validation
  // failure — not a synchronous throw out of the func call itself.
  const preparePath = hasPath ? [`    const path = ${url}`] : []
  // Maps the request body/query object to the wire and, when validation is on,
  // checks the actual snake_case payload against the strict wire schema before
  // sending. Runs inside the request() closure so a failure becomes Result.error,
  // not a synchronous throw — query params get the same guarantee as bodies do.
  const bodyValue = hasPath || hasQuery ? 'req.body' : 'req'
  const prepareBody = op.hasBody
    ? [
        `    const body = toWire(${bodyValue}, schemas.${op.funcName}Body)`,
        `    if (client._options.validate) {`,
        `      assertValid(schemas.${op.funcName}BodyWire, body)`,
        `    }`,
      ]
    : []
  const prepareQuery = hasQuery
    ? [
        // The query object is camelCase (sort pre-encoded to its wire string);
        // toWire snake-ifies the keys, including typed filter field names, against
        // the query schema. Record keys (label/dimension names) are preserved by
        // the walker.
        `    const query = toWire({`,
        ...op.queryParams.map((p) => {
          const key = toCamelCase(p)
          // sort.by names a field in the public (camelCase) surface; the server
          // expects the snake_case field name, so translate the value here.
          return p === 'sort'
            ? `      sort: encodeSort(req.sort, toSnakeCase),`
            : `      ${key}: req.${key},`
        }),
        `    }, schemas.${op.funcName}QueryParams)`,
        `    if (client._options.validate) {`,
        `      assertValid(schemas.${op.funcName}QueryParamsWire, query)`,
        `    }`,
        `    const searchParams = toURLSearchParams(query)`,
      ]
    : []
  const prepare = [...preparePath, ...prepareBody, ...prepareQuery]
  // An op with a path param, body, and/or query object runs inside a block-arrow
  // request() closure; a plain op (no path/body/query) keeps the terser
  // expression form (no behavior change).
  const open =
    prepare.length > 0 ? `  return request(() => {` : `  return request(() =>`
  const ret =
    prepare.length > 0 ? `    return http(client)` : `    http(client)`
  const closeBlock = prepare.length > 0
  if (op.hasResponse) {
    if (op.textResponseContentType) {
      lines.push(
        open,
        ...prepare,
        ret,
        `      .${op.verb}(${target}${optsArg})`,
        closeBlock ? `      .text()` : `      .text(),`,
        closeBlock ? `  })` : `  )`,
        `}`,
      )
    } else {
      // When validation is on, the raw snake_case wire response is checked against
      // the strict wire schema before fromWire maps it to the camelCase public shape.
      lines.push(
        open,
        ...prepare,
        ret,
        `      .${op.verb}(${target}${optsArg})`,
        `      .json()`,
        `      .then((data) => {`,
        `        if (client._options.validate) {`,
        `          assertValid(schemas.${op.funcName}ResponseWire, data)`,
        `        }`,
        `        return fromWire(data, schemas.${op.funcName}Response)`,
        closeBlock ? `      })` : `      }),`,
        closeBlock ? `  })` : `  )`,
        `}`,
      )
    }
  } else {
    // Bodyless/query-less void ops already used an async block; ops that prepare
    // a body or query add validation into the same block.
    lines.push(
      `  return request(async () => {`,
      ...prepare,
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
  // toWire maps request bodies and query objects to the wire; fromWire maps JSON
  // responses back. Both reference per-op schema values, so the schema namespace
  // is imported whenever either is used.
  const usesToWire = ops.some((op) => op.hasBody || op.queryParams.length > 0)
  const usesFromWire = ops.some(
    (op) => op.hasResponse && !op.textResponseContentType,
  )
  const usesSnake = ops.some((op) => op.hasSort)
  // assertValid runs (when the validate option is on) for any op with a request
  // body, query params, or a JSON response.
  const usesValidate = ops.some(
    (op) =>
      op.hasBody ||
      op.queryParams.length > 0 ||
      (op.hasResponse && !op.textResponseContentType),
  )
  const mapperNames = [
    ...(usesToWire ? ['toWire'] : []),
    ...(usesFromWire ? ['fromWire'] : []),
    ...(usesValidate ? ['assertValid'] : []),
    ...(usesSnake ? ['toSnakeCase'] : []),
  ]
  // The funcs reference only the per-op `…Request`/`…Response` aliases, which
  // live in the operations types module.
  // Path params are now inlined as template literals, so encodePath is no longer
  // imported; only query ops use the URL serializer and sort encoder.
  const hasAnyQuery = ops.some((op) => op.queryParams.length > 0)
  const imports = [
    `import { type Client, http } from '../core.js'`,
    `import { type Result, type RequestOptions } from '../lib/types.js'`,
    `import { request } from '../lib/request.js'`,
    ...(hasAnyQuery
      ? [`import { toURLSearchParams, encodeSort } from '../lib/encodings.js'`]
      : []),
    ...(mapperNames.length > 0
      ? [`import { ${mapperNames.join(', ')} } from '../lib/wire.js'`]
      : []),
    ...(mapperNames.length > 0
      ? [`import * as schemas from '../models/schemas.js'`]
      : []),
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
    `export { ValidationError, DepthLimitExceededError } from './lib/wire.js'`,
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
