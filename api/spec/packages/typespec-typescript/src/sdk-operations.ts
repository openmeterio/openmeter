import { type Operation, type Program, type Type } from '@typespec/compiler'
import { $ } from '@typespec/compiler/typekit'
import { getAllHttpServices, type HttpStatusCodesEntry } from '@typespec/http'
import { getOperationId } from '@typespec/openapi'
import { operationBaseName } from './ZodOperations.jsx'
import { shouldReference } from './utils.jsx'

export interface SdkOperation {
  funcName: string
  methodName: string
  base: string
  verb: string
  path: string
  pathParams: string[]
  queryParams: string[]
  hasSort: boolean
  hasBody: boolean
  hasResponse: boolean
  /** Documented interface name for the success body, when it is a named model. */
  responseInterface?: string
  /** The operation's `@doc`, for the README operations listing. */
  summary?: string
  /**
   * Documented interface name for the request body — the model's `…Input`
   * variant when its input shape diverges from its output, else the model
   * interface itself.
   */
  requestBodyInterface?: string
  nestPath: string[]
}

/** Resolves a type to its documented interface name (for response wiring). */
export type ResolveInterface = (type: Type | undefined) => string | undefined

/**
 * Resolves a request body type to the interface name a request should reference:
 * the input variant when the body diverges on input, else the output interface.
 */
export type ResolveRequestBody = (type: Type | undefined) => string | undefined

function lowerFirst(name: string): string {
  const m = name.match(/^([A-Z]{2,})([A-Z][a-z].*)$/)
  if (m?.[1] && m[2]) {
    return m[1].toLowerCase() + m[2]
  }
  return name.charAt(0).toLowerCase() + name.slice(1)
}

const NOISE_TOKENS = new Set(['metering'])

/** Split a resource name into lowercase words on separators and camelCase
 * boundaries, so a PascalCase resource like `PlanAddons` yields `plan` and
 * `addons`, and an acronym-prefixed one like `LLMCost` yields `llm` and `cost`
 * (not a single token). */
function resourceWords(resource: string): string[] {
  return resource
    .replace(/([A-Z]+)([A-Z][a-z])/g, '$1 $2')
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .split(/[\s\-_]+/)
    .filter(Boolean)
    .map((w) => w.toLowerCase())
}

function resourceTokens(resource: string): Set<string> {
  const tokens = new Set(NOISE_TOKENS)
  for (const lower of resourceWords(resource)) {
    tokens.add(lower)
    if (lower.endsWith('s')) {
      tokens.add(lower.slice(0, -1))
    } else {
      tokens.add(`${lower}s`)
    }
  }
  return tokens
}

function methodNameOf(
  operationId: string,
  resource: string,
  nestPath: string[],
): string {
  const strip = resourceTokens(resource)
  for (const seg of nestPath) {
    const lower = seg.toLowerCase()
    strip.add(lower)
    strip.add(lower.endsWith('s') ? lower.slice(0, -1) : `${lower}s`)
  }
  const parts = operationId.split(/[-_/\s]+/).filter(Boolean)
  const kept = parts.filter((p) => !strip.has(p.toLowerCase()))
  const used = kept.length > 0 ? kept : parts
  return used
    .map((p, i) =>
      i === 0 ? p.toLowerCase() : p.charAt(0).toUpperCase() + p.slice(1),
    )
    .join('')
}

export function sdkOperation(
  program: Program,
  op: Operation,
  resource: string,
  resolveInterface: ResolveInterface,
  resolveRequestBody: ResolveRequestBody,
): SdkOperation {
  const tk = $(program)
  const httpOp = tk.httpOperation.get(op)
  const base = operationBaseName(program, op)
  const operationId = getOperationId(program, op) ?? op.name

  const pathParams: string[] = []
  const queryParams: string[] = []
  for (const param of httpOp.parameters.parameters) {
    if (param.type === 'path') {
      pathParams.push(param.name)
    } else if (param.type === 'query') {
      queryParams.push(param.name)
    }
  }

  const { chain } = sourceOf(op)
  const nestPath = SPLIT_BY_INTERFACE.has(chain[0] ?? '')
    ? []
    : chain.slice(1).map((n) => lowerFirst(n))

  const responseBody = successBodyType(program, op)
  // A list endpoint's body is anonymous after HTTP extraction strips the envelope
  // identity, so it has no documented interface on its own. The response envelope
  // keeps its `@friendlyName` through extraction
  // (`PagePaginatedResponse<Meter>` -> `MeterPagePaginatedResponse`), recovering one.
  const responseInterface =
    (responseBody && shouldReference(program, responseBody)
      ? resolveInterface(responseBody)
      : undefined) ?? resolveInterface(successResponseEnvelope(program, op))

  const requestBodyInterface = resolveRequestBody(httpOp.parameters.body?.type)

  return {
    funcName: lowerFirst(base),
    methodName: methodNameOf(operationId, resource, nestPath),
    base,
    verb: httpOp.verb,
    path: httpOp.path,
    pathParams,
    queryParams,
    nestPath,
    hasSort: queryParams.includes('sort'),
    hasBody: httpOp.parameters.body?.type !== undefined,
    hasResponse: responseBody !== undefined,
    requestBodyInterface,
    responseInterface,
    summary: tk.type.getDoc(op),
  }
}

function is2xx(status: HttpStatusCodesEntry): boolean {
  return (
    status === '*' ||
    (typeof status === 'number' && status >= 200 && status < 300) ||
    (typeof status === 'object' && status.start >= 200 && status.start < 300)
  )
}

function successBodyType(program: Program, op: Operation): Type | undefined {
  const httpOp = $(program).httpOperation.get(op)
  for (const response of httpOp.responses) {
    if (!is2xx(response.statusCodes)) {
      continue
    }
    for (const r of response.responses) {
      if (r.body?.type) {
        return r.body.type
      }
    }
  }
  return undefined
}

// The success response envelope retains its declared identity (and
// `@friendlyName`) where the extracted body does not, so it recovers a
// documented interface for responses whose body is anonymous after extraction.
function successResponseEnvelope(
  program: Program,
  op: Operation,
): Type | undefined {
  const httpOp = $(program).httpOperation.get(op)
  for (const response of httpOp.responses) {
    if (is2xx(response.statusCodes)) {
      return response.type
    }
  }
  return undefined
}

// Namespaces split into one client per interface instead of one per namespace.
const SPLIT_BY_INTERFACE = new Set(['ProductCatalog'])

function interfaceResource(interfaceName: string): string {
  return pluralize(interfaceName.replace(/Operations$/, ''))
}

// The op we walk lives on an `*Endpoints` interface whose own namespace is
// `OpenMeter`; the meaningful grouping is on the interface it `extends`.
function sourceOf(op: Operation): {
  chain: string[]
  interface?: string
} {
  const source = op.interface?.sourceInterfaces?.[0]
  const chain: string[] = []
  for (let ns = source?.namespace; ns && ns.name; ns = ns.namespace) {
    chain.unshift(ns.name)
  }
  return { chain, interface: source?.name }
}

export function groupOperations(
  operations: Operation[],
): Map<string, Operation[]> {
  const groups = new Map<string, Operation[]>()
  for (const op of operations) {
    const { chain, interface: iface } = sourceOf(op)
    const top = chain[0]
    if (!top) {
      continue
    }
    const key =
      SPLIT_BY_INTERFACE.has(top) && iface ? interfaceResource(iface) : top
    const existing = groups.get(key)
    if (existing) {
      existing.push(op)
    } else {
      groups.set(key, [op])
    }
  }
  return groups
}

/**
 * Request body overrides keyed by operation base name. A `@sharedRoute`
 * endpoint declares one operation per content type but is collapsed to a single
 * SDK operation that keeps the first variant (for its doc, summary, and
 * response). When a sibling variant carries the `application/json` body and its
 * type differs from the kept variant's, the SDK should accept that body — it is
 * the JSON shape a client sends. This maps such an endpoint to its
 * `application/json` body type so the request type can render the full shape
 * (e.g. the single-or-batch ingest union) instead of only the first variant's.
 */
export function jsonBodyOverrides(program: Program): Map<string, Type> {
  const tk = $(program)
  const [services] = getAllHttpServices(program)
  const firstBody = new Map<string, Type | undefined>()
  const jsonBody = new Map<string, Type>()
  for (const service of services) {
    for (const httpOp of service.operations) {
      const base = operationBaseName(program, httpOp.operation)
      const body = tk.httpOperation.get(httpOp.operation).parameters.body
      if (!firstBody.has(base)) {
        firstBody.set(base, body?.type)
      }
      if (body?.contentTypes.includes('application/json') && body.type) {
        jsonBody.set(base, body.type)
      }
    }
  }
  const overrides = new Map<string, Type>()
  for (const [base, body] of jsonBody) {
    if (body !== firstBody.get(base)) {
      overrides.set(base, body)
    }
  }
  return overrides
}

export function pluralize(word: string): string {
  if (word.endsWith('s')) {
    return word
  }
  if (/(x|z|ch|sh)$/i.test(word)) {
    return `${word}es`
  }
  if (/[^aeiou]y$/i.test(word)) {
    return `${word.slice(0, -1)}ies`
  }
  return `${word}s`
}

export function namespaceNames(resource: string): {
  class: string
  getter: string
} {
  const cls = resource.charAt(0).toUpperCase() + resource.slice(1)
  return { class: cls, getter: lowerFirst(cls) }
}
