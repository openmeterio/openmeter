import {
  getFriendlyName,
  type Model,
  type ModelProperty,
  type Operation,
  type Program,
  type Type,
} from '@typespec/compiler'
import { $ } from '@typespec/compiler/typekit'
import {
  getAllHttpServices,
  type HttpOperationParameter,
  type HttpOperationQueryParameter,
  type HttpStatusCodesEntry,
} from '@typespec/http'
import '@typespec/http/experimental/typekit'
import { getOperationId } from '@typespec/openapi'
import { optionalTypeName } from './go-types.js'
import { methodNameFromOperationName, methodNameOf } from './grouping.js'

export type GoQueryCodec =
  | { kind: 'page' }
  | { kind: 'cursorPage' }
  | { kind: 'sort' }
  | { kind: 'deepObject'; model: Model }
  | { kind: 'array'; explode: boolean }
  | { kind: 'scalar' }

export interface GoOperation {
  operation: Operation
  methodName: string
  verb: string
  path: string
  pathParams: GoParameter[]
  queryParams: GoParameter[]
  body?: Type
  bodyOptional: boolean
  requestContentType?: string
  response?: Type
  responseContentType?: string
  pagination?: 'page' | 'cursor'
}

export interface GoParameter {
  name: string
  property: ModelProperty
  type: Type
  queryCodec?: GoQueryCodec
}

export function collectHttpOperations(
  program: Program,
  includeServices?: string[],
): Operation[] {
  const [services] = getAllHttpServices(program)
  const included =
    includeServices && includeServices.length > 0
      ? services.filter((service) =>
          includeServices.includes(service.namespace.name),
        )
      : services
  const seen = new Set<string>()
  const operations: Operation[] = []

  for (const service of included) {
    for (const httpOperation of service.operations) {
      const operation = httpOperation.operation
      const identity = operationRepresentationKey(program, operation)
      if (seen.has(identity)) {
        continue
      }
      seen.add(identity)
      operations.push(operation)
    }
  }

  return operations
}

export function describeOperations(
  program: Program,
  resource: string,
  operations: Operation[],
  bodyOverrides: Map<string, Type> = new Map(),
  nestPath: string[] = [],
): GoOperation[] {
  const typekit = $(program)

  const described = operations.map((operation) => {
    const httpOperation = typekit.httpOperation.get(operation)
    assertSupportedParameters(
      operation,
      httpOperation.parameters.parameters,
      httpOperation.parameters.body?.contentTypeProperty,
    )
    if (
      httpOperation.parameters.body &&
      httpOperation.parameters.body.bodyKind !== 'single'
    ) {
      throw new Error(
        `typespec-go: unsupported ${httpOperation.parameters.body.bodyKind} request body on ${operation.name}; add an explicit Go body codec before emitting it`,
      )
    }
    const pathParams = httpOperation.parameters.parameters
      .filter((parameter) => parameter.type === 'path')
      .map((parameter) => ({
        name: parameter.name,
        property: parameter.param,
        type: parameter.param.type,
      }))
    const queryParams = httpOperation.parameters.parameters
      .filter((parameter) => parameter.type === 'query')
      .map((parameter) => ({
        name: parameter.name,
        property: parameter.param,
        type: parameter.param.type,
        queryCodec: classifyQueryParameter(program, operation, parameter),
      }))
    const response = successBody(program, operation, httpOperation.responses)
    const overrideKey = qualifiedOperationKey(operation)
    const body =
      bodyOverrides.get(overrideKey) ?? httpOperation.parameters.body?.type
    const requestContentType = bodyOverrides.has(overrideKey)
      ? 'application/json'
      : preferredContentType(httpOperation.parameters.body?.contentTypes)

    return {
      operation,
      methodName: methodNameOf(program, operation, resource, nestPath),
      verb: httpOperation.verb,
      path: httpOperation.path,
      pathParams,
      queryParams,
      body,
      bodyOptional:
        body !== undefined &&
        !bodyOverrides.has(overrideKey) &&
        (httpOperation.parameters.body?.property?.optional ?? false),
      requestContentType,
      response:
        response &&
        (optionalTypeName(program, response.type)
          ? response.type
          : response.envelope),
      responseContentType: response?.contentType,
      pagination: paginationKind(queryParams),
    }
  })

  return disambiguateMethodNames(described)
}

export function classifyQueryParameter(
  program: Program,
  operation: Operation,
  parameter: HttpOperationQueryParameter,
): GoQueryCodec {
  const typekit = $(program)
  const type = parameter.param.type
  if (parameter.name === 'sort') {
    return { kind: 'sort' }
  }

  if (type.kind === 'Model') {
    if (typekit.array.is(type)) {
      return { kind: 'array', explode: parameter.explode }
    }

    if (parameter.style === 'deepObject') {
      // Only a parameter literally named `page` may become a pagination
      // codec, and its property set must match one pagination shape exactly;
      // otherwise a coincidentally shaped filter model would be silently
      // reclassified and lose its remaining properties.
      if (parameter.name !== 'page') {
        return { kind: 'deepObject', model: type }
      }

      const properties = new Set(type.properties.keys())
      const within = (allowed: readonly string[]) =>
        [...properties].every((key) => allowed.includes(key))
      if (
        (properties.has('after') || properties.has('before')) &&
        within(['size', 'after', 'before'])
      ) {
        return { kind: 'cursorPage' }
      }
      if (properties.has('number') && within(['size', 'number'])) {
        return { kind: 'page' }
      }

      throw new Error(
        `typespec-go: query parameter page on ${operation.name} has properties {${[...properties].join(', ')}} matching neither page pagination {size, number} nor cursor pagination {size, after, before}; rename the parameter or align it with a pagination shape before emitting it`,
      )
    }
  }

  return { kind: 'scalar' }
}

/**
 * Request body overrides for @sharedRoute siblings.
 *
 * TypeSpec exposes the CSV meter query as a response-only sibling of the JSON
 * query. Both operations share an operation id, so the JSON sibling's request
 * body is also the request body for the CSV method.
 *
 * Shared-route variants that already declare their own body keep that body and
 * content type. That lets the Go SDK expose explicit media-type-specific
 * overloads such as CloudEvents single-event, CloudEvents batch, and generic
 * application/json ingest.
 *
 * Siblings are associated by their shared route (declaring container + verb +
 * path) rather than by operation id or name, so two unrelated operations that
 * happen to share a name in different namespaces can never donate a body to
 * each other. The returned map is keyed by qualifiedOperationKey.
 */
export function jsonBodyOverrides(program: Program): Map<string, Type> {
  const typekit = $(program)
  const [services] = getAllHttpServices(program)
  const bodylessRoutes = new Map<string, string>()
  const jsonBodyByRoute = new Map<string, Type>()

  for (const service of services) {
    for (const httpOperation of service.operations) {
      const operation = httpOperation.operation
      const routeKey = [
        containerPath(operation).join('.'),
        httpOperation.verb,
        httpOperation.path,
      ].join('|')
      const body = typekit.httpOperation.get(operation).parameters.body

      if (!body?.type) {
        bodylessRoutes.set(qualifiedOperationKey(operation), routeKey)
      } else if (body.contentTypes.includes('application/json')) {
        jsonBodyByRoute.set(routeKey, body.type)
      }
    }
  }

  const overrides = new Map<string, Type>()
  for (const [operationKey, routeKey] of bodylessRoutes) {
    const json = jsonBodyByRoute.get(routeKey)
    if (json) {
      overrides.set(operationKey, json)
    }
  }

  return overrides
}

function qualifiedOperationKey(operation: Operation): string {
  return [...containerPath(operation), operation.name].join('.')
}

function containerPath(operation: Operation): string[] {
  const path: string[] = []
  for (
    let namespace = operation.interface?.namespace ?? operation.namespace;
    namespace?.name;
    namespace = namespace.namespace
  ) {
    path.unshift(namespace.name)
  }
  if (operation.interface) {
    path.push(operation.interface.name)
  }

  return path
}

export function operationBaseName(
  program: Program,
  operation: Operation,
): string {
  const identity =
    getFriendlyName(program, operation) ??
    getOperationId(program, operation) ??
    operation.name

  return identity
    .split(/[-_/\s]+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join('')
}

function operationRepresentationKey(
  program: Program,
  operation: Operation,
): string {
  const httpOperation = $(program).httpOperation.get(operation)
  const body = httpOperation.parameters.body
  const response = successBody(program, operation, httpOperation.responses)

  return [
    operationBaseName(program, operation),
    httpOperation.verb,
    httpOperation.path,
    body?.contentTypes.join(',') ?? '',
    typeKey(program, body?.type),
    response?.contentType ?? '',
    typeKey(program, response?.type),
  ].join('|')
}

function typeKey(
  program: Program,
  type: Type | undefined,
  seen = new Set<Type>(),
): string {
  if (!type) {
    return ''
  }

  const named = optionalTypeName(program, type)
  if (named) {
    return `${type.kind}:${named}`
  }

  if (seen.has(type)) {
    return type.kind
  }
  seen.add(type)

  switch (type.kind) {
    case 'Model': {
      const typekit = $(program)
      if (typekit.array.is(type)) {
        return `array:${typeKey(program, type.indexer?.value, seen)}`
      }
      if (typekit.record.is(type)) {
        return `record:${typeKey(program, type.indexer?.value, seen)}`
      }
      return `model:{${[...type.properties.values()]
        .map(
          (property) =>
            `${property.name}:${typeKey(program, property.type, seen)}`,
        )
        .join(',')}}`
    }
    case 'Union':
      return `union:${[...type.variants.values()]
        .map((variant) => typeKey(program, variant.type, seen))
        .join('|')}`
    case 'Tuple':
      return `tuple:${type.values
        .map((value) => typeKey(program, value, seen))
        .join('|')}`
    default:
      return `${type.kind}:${'value' in type ? String(type.value) : ''}`
  }
}

function disambiguateMethodNames(operations: GoOperation[]): GoOperation[] {
  const byMethodName = new Map<string, GoOperation[]>()
  for (const operation of operations) {
    const group = byMethodName.get(operation.methodName)
    if (group) {
      group.push(operation)
    } else {
      byMethodName.set(operation.methodName, [operation])
    }
  }

  for (const group of byMethodName.values()) {
    if (group.length < 2) {
      continue
    }

    for (const operation of group) {
      operation.methodName = methodNameFromOperationName(
        operation.operation.name,
      )
    }
  }

  return operations
}

function assertSupportedParameters(
  operation: Operation,
  parameters: HttpOperationParameter[],
  contentTypeProperty: ModelProperty | undefined,
): void {
  for (const parameter of parameters) {
    if (parameter.type === 'path' || parameter.type === 'query') {
      continue
    }
    if (
      parameter.type === 'header' &&
      contentTypeProperty &&
      parameter.param === contentTypeProperty
    ) {
      continue
    }

    throw new Error(
      `typespec-go: unsupported ${parameter.type} parameter ${parameter.name} on ${operation.name}; add an explicit Go ${parameter.type} codec before emitting it`,
    )
  }
}

function is2xx(status: HttpStatusCodesEntry): boolean {
  return (
    status === '*' ||
    (typeof status === 'number' && status >= 200 && status < 300) ||
    (typeof status === 'object' && status.start >= 200 && status.start < 300)
  )
}

function successBody(
  program: Program,
  operation: Operation,
  responses: ReturnType<
    ReturnType<typeof $>['httpOperation']['get']
  >['responses'],
): { type: Type; envelope: Type; contentType?: string } | undefined {
  let found: { type: Type; envelope: Type; contentType?: string } | undefined
  let foundKey: string | undefined

  for (const response of responses) {
    if (!is2xx(response.statusCodes)) {
      continue
    }
    for (const content of response.responses) {
      if (!content.body?.type) {
        continue
      }
      if (found === undefined) {
        found = {
          type: content.body.type,
          envelope: response.type,
          contentType: content.body.contentTypes[0],
        }
        foundKey = typeKey(program, content.body.type)
      } else if (foundKey !== typeKey(program, content.body.type)) {
        const describe = (type: Type) =>
          optionalTypeName(program, type) ?? type.kind
        throw new Error(
          `typespec-go: operation ${operation.name} declares multiple 2xx response bodies with different types (${describe(found.type)} vs ${describe(content.body.type)}); split the variants into @sharedRoute siblings or align the response models before emitting it`,
        )
      }
    }
  }

  return found
}

function preferredContentType(
  contentTypes: readonly string[] | undefined,
): string | undefined {
  return (
    contentTypes?.find((contentType) => contentType === 'application/json') ??
    contentTypes?.[0]
  )
}

function paginationKind(queryParams: GoParameter[]): GoOperation['pagination'] {
  for (const parameter of queryParams) {
    if (parameter.queryCodec?.kind === 'page') {
      return 'page'
    }
    if (parameter.queryCodec?.kind === 'cursorPage') {
      return 'cursor'
    }
  }

  return undefined
}
