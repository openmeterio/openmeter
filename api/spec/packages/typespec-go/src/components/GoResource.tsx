import * as ay from '@alloy-js/core'
import * as go from '@alloy-js/go'
import type { Refkey } from '@alloy-js/core'
import {
  resolveEncodedName,
  walkPropertiesInherited,
  type ModelProperty,
  type Operation,
  type Program,
  type Type,
} from '@typespec/compiler'
import { $ } from '@typespec/compiler/typekit'
import { isHeader, isStatusCode } from '@typespec/http'
import {
  goExportedName,
  goType,
  queryFilterKind,
  queryScalarKind,
  typeName,
} from '../go-types.js'
import {
  describeOperations,
  operationBaseName,
  type GoOperation,
  type GoParameter,
} from '../operations.js'
import { context, iter, strconv, strings, url } from '../stdlib.js'
import { GoStruct } from './GoStruct.js'

export interface GoResourceProps {
  program: Program
  resource: string
  serviceName: string
  nestPath: string[]
  operations: Operation[]
  bodyOverrides: Map<string, Type>
  serviceRefkey: Refkey
  children: Array<{ name: string; serviceRefkey: Refkey }>
}

export function GoResource({
  program,
  resource,
  serviceName,
  nestPath,
  operations,
  bodyOverrides,
  serviceRefkey,
  children,
}: GoResourceProps) {
  const described = describeOperations(
    program,
    resource,
    operations,
    bodyOverrides,
    nestPath,
  )
  const paramsNames = resolveListParamsNames(program, described)
  const params = listParameterDeclarations(program, described, paramsNames)

  return (
    <ay.List joiner={'\n\n'}>
      <go.StructTypeDeclaration
        name={`${goExportedName(serviceName)}Service`}
        refkey={serviceRefkey}
      >
        <ay.List hardline>
          <go.StructMember name="client" type="*Client" />
          <ay.List hardline>
            {children.map((child) => (
              <go.StructMember
                name={goExportedName(child.name)}
                type={
                  <go.Pointer>
                    <go.Reference refkey={child.serviceRefkey} />
                  </go.Pointer>
                }
              />
            ))}
          </ay.List>
        </ay.List>
      </go.StructTypeDeclaration>
      {params}
      <ay.List>
        {described.flatMap((operation) => [
          <OperationMethod
            program={program}
            operation={operation}
            serviceRefkey={serviceRefkey}
            paramsNames={paramsNames}
          />,
          '\n\n',
          operation.pagination &&
          isPlainPageEnvelope(program, operation.response) ? (
            <>
              <ListAllMethod
                program={program}
                operation={operation}
                serviceRefkey={serviceRefkey}
                paramsNames={paramsNames}
              />
              {'\n\n'}
            </>
          ) : undefined,
          isTextResponse(operation) ? (
            <>
              <StreamMethod
                program={program}
                operation={operation}
                serviceRefkey={serviceRefkey}
                paramsNames={paramsNames}
              />
              {'\n\n'}
            </>
          ) : undefined,
        ])}
      </ay.List>
    </ay.List>
  )
}

function OperationMethod({
  program,
  operation,
  serviceRefkey,
  paramsNames,
}: {
  program: Program
  operation: GoOperation
  serviceRefkey: Refkey
  paramsNames: Map<GoOperation, string>
}) {
  // goType resolves the read-side reference name, honoring structural-dedupe
  // aliases; typeName alone would reference a collapsed declaration.
  const responseName = operation.response
    ? readReferenceName(program, operation.response)
    : undefined
  const textResponse = isTextResponse(operation)
  const parameters = methodParameters(program, operation, paramsNames)
  const path = pathCode(operation)
  const requestBody = operation.body
    ? operation.bodyOptional
      ? 'optionalBody(request)'
      : 'request'
    : 'nil'
  const query = operation.queryParams.length > 0 ? 'params.values()' : 'nil'
  const requestContentType = operation.body
    ? JSON.stringify(operation.requestContentType ?? 'application/json')
    : '""'
  const accept = JSON.stringify(
    operation.responseContentType ??
      (operation.response ? 'application/json' : ''),
  )
  const returns = textResponse
    ? ['[]byte', 'error']
    : responseName
      ? [`*${responseName}`, 'error']
      : 'error'

  return (
    <go.FunctionDeclaration
      name={goExportedName(operation.methodName)}
      receiver={serviceReceiver(serviceRefkey)}
      parameters={parameters}
      returns={returns}
      doc={$(program).type.getDoc(operation.operation)}
    >
      {textResponse
        ? ay.code`
            ${path}

            req, err := s.client.newRequestWithContentType(ctx, ${httpMethod(operation.verb)}, path, ${query}, ${requestBody}, ${requestContentType}, ${accept})
            if err != nil {
              return nil, err
            }

            return s.client.doRaw(req)
          `
        : responseName
          ? ay.code`
            ${path}

            req, err := s.client.newRequestWithContentType(ctx, ${httpMethod(operation.verb)}, path, ${query}, ${requestBody}, ${requestContentType}, ${accept})
            if err != nil {
              return nil, err
            }

            var out ${responseName}
            if err := s.client.doJSON(req, &out); err != nil {
              return nil, err
            }

            return &out, nil
          `
          : ay.code`
              ${path}

              req, err := s.client.newRequestWithContentType(ctx, ${httpMethod(operation.verb)}, path, ${query}, ${requestBody}, ${requestContentType}, ${accept})
              if err != nil {
                return err
              }

              _, err = s.client.doRaw(req)
              return err
            `}
    </go.FunctionDeclaration>
  )
}

function ListAllMethod({
  program,
  operation,
  serviceRefkey,
  paramsNames,
}: {
  program: Program
  operation: GoOperation
  serviceRefkey: Refkey
  paramsNames: Map<GoOperation, string>
}) {
  if (operation.pagination === 'cursor') {
    return (
      <CursorListAllMethod
        program={program}
        operation={operation}
        serviceRefkey={serviceRefkey}
        paramsNames={paramsNames}
      />
    )
  }

  const element = pageElement(program, operation.response!)
  const listArguments = [
    ...operation.pathParams.map((parameter) => localName(parameter.name)),
    ...(operation.body ? ['request'] : []),
    'pageParams',
  ].join(', ')

  return (
    <go.FunctionDeclaration
      name={`${goExportedName(operation.methodName)}All`}
      receiver={serviceReceiver(serviceRefkey)}
      parameters={methodParameters(program, operation, paramsNames)}
      returns={<Seq2Type element={element} />}
      doc={allIteratorDoc(operation, element)}
    >
      {ay.code`
        return paginate(params.Page, func(page, size int) ([]${element}, int, error) {
          pageParams := params
          pageParams.Page = &PageParams{Size: Int(size), Number: Int(page)}

          resp, err := s.${goExportedName(operation.methodName)}(ctx, ${listArguments})
          if err != nil {
            return nil, 0, err
          }

          return resp.Data, resp.Meta.Page.Total, nil
        })
      `}
    </go.FunctionDeclaration>
  )
}

function CursorListAllMethod({
  program,
  operation,
  serviceRefkey,
  paramsNames,
}: {
  program: Program
  operation: GoOperation
  serviceRefkey: Refkey
  paramsNames: Map<GoOperation, string>
}) {
  const element = pageElement(program, operation.response!)
  const listArguments = [
    ...operation.pathParams.map((parameter) => localName(parameter.name)),
    ...(operation.body ? ['request'] : []),
    'pageParams',
  ].join(', ')

  return (
    <go.FunctionDeclaration
      name={`${goExportedName(operation.methodName)}All`}
      receiver={serviceReceiver(serviceRefkey)}
      parameters={methodParameters(program, operation, paramsNames)}
      returns={<Seq2Type element={element} />}
      doc={allIteratorDoc(operation, element)}
    >
      {ay.code`
        return paginateCursor(params.Page, func(after, before *string, size int) ([]${element}, *string, *string, error) {
          pageParams := params
          pageParams.Page = &CursorPageParams{Size: Int(size), After: after, Before: before}

          resp, err := s.${goExportedName(operation.methodName)}(ctx, ${listArguments})
          if err != nil {
            return nil, nil, nil, err
          }

          return resp.Data, String(resp.Meta.Page.Next.GetOrEmpty()), String(resp.Meta.Page.Previous.GetOrEmpty()), nil
        })
      `}
    </go.FunctionDeclaration>
  )
}

function StreamMethod({
  program,
  operation,
  serviceRefkey,
  paramsNames,
}: {
  program: Program
  operation: GoOperation
  serviceRefkey: Refkey
  paramsNames: Map<GoOperation, string>
}) {
  const requestBody = operation.body
    ? operation.bodyOptional
      ? 'optionalBody(request)'
      : 'request'
    : 'nil'

  return (
    <go.FunctionDeclaration
      name={`${goExportedName(operation.methodName)}Stream`}
      receiver={serviceReceiver(serviceRefkey)}
      parameters={methodParameters(program, operation, paramsNames)}
      returns={[go.std.io.ReadCloser, 'error']}
    >
      {ay.code`
        ${pathCode(operation)}

        req, err := s.client.newRequestWithContentType(ctx, ${httpMethod(operation.verb)}, path, ${operation.queryParams.length > 0 ? 'params.values()' : 'nil'}, ${requestBody}, ${JSON.stringify(operation.requestContentType ?? 'application/json')}, ${JSON.stringify(operation.responseContentType ?? 'text/csv')})
        if err != nil {
          return nil, err
        }

        resp, err := s.client.doStream(req)
        if err != nil {
          return nil, err
        }

        return resp.Body, nil
      `}
    </go.FunctionDeclaration>
  )
}

function listParameterDeclarations(
  program: Program,
  operations: GoOperation[],
  paramsNames: Map<GoOperation, string>,
) {
  const emitted = new Set<string>()

  return (
    <ay.List joiner={'\n\n'}>
      {operations.flatMap((operation) => {
        if (operation.queryParams.length === 0) {
          return []
        }

        // resolveListParamsNames guarantees operations sharing a params name
        // also share a query shape, so deduping by name emits one identical
        // struct for all of them.
        const paramsName = paramsNames.get(operation)!
        if (emitted.has(paramsName)) {
          return []
        }
        emitted.add(paramsName)

        const deepObjects = operation.queryParams.filter(
          (parameter) => parameter.queryCodec?.kind === 'deepObject',
        )
        const deepObjectNames = new Map(
          deepObjects.map((parameter) => [
            parameter.name,
            deepObjectName(paramsName, parameter),
          ]),
        )
        const paramsRefkey = ay.refkey()

        return [
          <ay.List joiner={'\n\n'}>
            {deepObjects.map((parameter) => {
              const codec = parameter.queryCodec!
              if (codec.kind !== 'deepObject') {
                return undefined
              }
              return (
                <GoStruct
                  name={deepObjectNames.get(parameter.name)!}
                  fields={[...codec.model.properties.values()].map((property) =>
                    queryFilterField(program, property),
                  )}
                  tags={false}
                />
              )
            })}
            <go.StructTypeDeclaration name={paramsName} refkey={paramsRefkey}>
              <ay.List hardline>
                {operation.queryParams.map((parameter) =>
                  queryParamField(program, parameter, deepObjectNames),
                )}
              </ay.List>
            </go.StructTypeDeclaration>
            <go.FunctionDeclaration
              name="values"
              receiver={<go.FunctionReceiver name="p" type={paramsRefkey} />}
              returns={url.Values}
            >
              {queryValuesBody(program, operation.queryParams)}
            </go.FunctionDeclaration>
          </ay.List>,
        ]
      })}
    </ay.List>
  )
}

function queryParamField(
  program: Program,
  parameter: GoParameter,
  deepObjectNames: Map<string, string>,
) {
  switch (parameter.queryCodec?.kind) {
    case 'page':
      return <go.StructMember name="Page" type="*PageParams" />
    case 'cursorPage':
      return <go.StructMember name="Page" type="*CursorPageParams" />
    case 'sort':
      return (
        <go.StructMember name={goExportedName(parameter.name)} type="*Sort" />
      )
    case 'deepObject':
      return (
        <go.StructMember
          name={goExportedName(parameter.name)}
          type={`*${deepObjectNames.get(parameter.name)!}`}
        />
      )
    default:
      const mapped = goType(program, parameter.type)
      return (
        <go.StructMember
          name={goExportedName(parameter.name)}
          type={
            parameter.property.optional && !mapped.nilable ? (
              <go.Pointer>{mapped.type}</go.Pointer>
            ) : (
              mapped.type
            )
          }
        />
      )
  }
}

function queryValuesBody(program: Program, parameters: GoParameter[]) {
  const statements: ay.Children[] = [ay.code`q := ${url.Values}{}`]

  for (const parameter of parameters) {
    switch (parameter.queryCodec?.kind) {
      case 'page':
        statements.push(ay.code`addPageParams(q, p.Page)`)
        break
      case 'cursorPage':
        statements.push(ay.code`addCursorPageParams(q, p.Page)`)
        break
      case 'sort':
        statements.push(
          ay.code`addSort(q, ${JSON.stringify(parameter.name)}, p.${goExportedName(parameter.name)})`,
        )
        break
      case 'deepObject':
        statements.push(deepObjectValues(program, parameter))
        break
      case 'array': {
        const field = `p.${goExportedName(parameter.name)}`
        const wireName = JSON.stringify(parameter.name)
        if (parameter.type.kind !== 'Model') {
          throw new Error(
            `array query parameter ${parameter.name} is not a model`,
          )
        }
        const element = parameter.type.indexer?.value
        if (!element) {
          throw new Error(
            `array query parameter ${parameter.name} has no element type`,
          )
        }
        const value = queryScalarValue(program, element, 'value')
        if (parameter.queryCodec.explode) {
          statements.push(ay.code`for _, value := range ${field} {
            q.Add(${wireName}, ${value})
          }`)
        } else {
          const values = `${localName(parameter.name)}Values`
          statements.push(ay.code`if len(${field}) > 0 {
            ${values} := make([]string, 0, len(${field}))
            for _, value := range ${field} {
              ${values} = append(${values}, ${value})
            }
            q.Set(${wireName}, ${strings.Join}(${values}, ","))
          }`)
        }
        break
      }
      case 'scalar':
      default: {
        const field = `p.${goExportedName(parameter.name)}`
        const wireName = JSON.stringify(parameter.name)
        if (parameter.property.optional) {
          const value = queryScalarValue(program, parameter.type, `*${field}`)
          statements.push(ay.code`if ${field} != nil {
            q.Set(${wireName}, ${value})
          }`)
        } else {
          const value = queryScalarValue(program, parameter.type, field)
          statements.push(ay.code`q.Set(${wireName}, ${value})`)
        }
        break
      }
    }
  }

  statements.push(ay.code`return q`)
  return <ay.List joiner={'\n\n'}>{statements}</ay.List>
}

export function deepObjectName(
  paramsName: string,
  parameter: GoParameter,
): string {
  if (parameter.name === 'filter') {
    return `${paramsName.replace(/ListParams$/, '').replace(/Params$/, '')}Filter`
  }

  return `${paramsName.replace(/Params$/, '')}${goExportedName(parameter.name)}`
}

function queryFilterField(program: Program, property: ModelProperty) {
  const kind = queryFilterKind(program, property.type)
  let type: ay.Children

  switch (kind) {
    case 'string':
      type = <go.Pointer>StringFilter</go.Pointer>
      break
    case 'stringExact':
      type = <go.Pointer>StringExactFilter</go.Pointer>
      break
    case 'dateTime':
      type = <go.Pointer>DateTimeFilter</go.Pointer>
      break
    case 'numeric':
      type = <go.Pointer>NumericFilter</go.Pointer>
      break
    case 'boolean':
      type = <go.Pointer>BooleanFilter</go.Pointer>
      break
    case 'labels':
      type = 'map[string]*StringFilter'
      break
    case 'scalar': {
      const mapped = goType(program, property.type)
      type = mapped.nilable ? (
        mapped.type
      ) : (
        <go.Pointer>{mapped.type}</go.Pointer>
      )
      break
    }
  }

  return {
    name: goExportedName(property.name),
    wireName: resolveEncodedName(program, property, 'application/json'),
    type,
    optional: false,
    nilable: false,
    doc: $(program).type.getDoc(property),
  }
}

function deepObjectValues(
  program: Program,
  parameter: GoParameter,
): ay.Children {
  const codec = parameter.queryCodec
  if (codec?.kind !== 'deepObject') {
    throw new Error(`${parameter.name} is not a deep-object query parameter`)
  }

  const root = `p.${goExportedName(parameter.name)}`
  const fields = [...codec.model.properties.values()].map((property) => {
    const field = `${root}.${goExportedName(property.name)}`
    const wireName = resolveEncodedName(program, property, 'application/json')
    const prefix = JSON.stringify(`${parameter.name}[${wireName}]`)

    switch (queryFilterKind(program, property.type)) {
      case 'string':
        return ay.code`addStringFilter(q, ${prefix}, ${field})`
      case 'stringExact':
        return ay.code`addStringExactFilter(q, ${prefix}, ${field})`
      case 'dateTime':
        return ay.code`addDateTimeFilter(q, ${prefix}, ${field})`
      case 'numeric':
        return ay.code`addNumericFilter(q, ${prefix}, ${field})`
      case 'boolean':
        return ay.code`addBooleanFilter(q, ${prefix}, ${field})`
      case 'labels':
        return ay.code`for key, filter := range ${field} {
          addStringFilter(q, ${JSON.stringify(`${parameter.name}[${wireName}][`)}+key+"]", filter)
        }`
      case 'scalar': {
        const value = queryScalarValue(program, property.type, `*${field}`)
        return ay.code`if ${field} != nil {
          q.Set(${prefix}, ${value})
        }`
      }
    }
  })

  return ay.code`if ${root} != nil {
    ${(<ay.List joiner={'\n'}>{fields}</ay.List>)}
  }`
}

export function queryScalarValue(
  program: Program,
  type: Type,
  expression: string,
): ay.Children {
  const mapped = goType(program, type).type
  const convert = (target: string) =>
    mapped === target ? expression : `${target}(${expression})`

  switch (queryScalarKind(program, type)) {
    case 'string':
      return convert('string')
    case 'boolean':
      return ay.code`${strconv.FormatBool}(${convert('bool')})`
    case 'integer':
      return isUnsignedType(type)
        ? ay.code`${strconv.FormatUint}(${convert('uint64')}, 10)`
        : ay.code`${strconv.FormatInt}(${convert('int64')}, 10)`
    case 'float':
      return ay.code`${strconv.FormatFloat}(${convert('float64')}, 'g', -1, 64)`
    case 'dateTime':
      // A dereference like *p.From must be parenthesized before .Format so the
      // selector does not bind tighter than the dereference.
      return ay.code`${expression.startsWith('*') ? `(${expression})` : expression}.Format(${go.std.time.RFC3339Nano})`
  }
}

function isUnsignedType(type: Type): boolean {
  if (type.kind !== 'Scalar') {
    return false
  }
  for (
    let current: typeof type | undefined = type;
    current;
    current = current.baseScalar
  ) {
    if (/^uint/.test(current.name)) {
      return true
    }
  }
  return false
}

function methodParameters(
  program: Program,
  operation: GoOperation,
  paramsNames: Map<GoOperation, string>,
): { name: string; type: ay.Children }[] {
  const parameters: { name: string; type: ay.Children }[] = [
    { name: 'ctx', type: context.Context },
    ...operation.pathParams.map((parameter) => ({
      name: localName(parameter.name),
      type: 'string',
    })),
  ]
  if (operation.body) {
    const mapped = goType(program, operation.body, { mode: 'input' })
    parameters.push({
      name: 'request',
      type:
        operation.bodyOptional && !mapped.nilable ? (
          <go.Pointer>{mapped.type}</go.Pointer>
        ) : (
          mapped.type
        ),
    })
  }
  if (operation.queryParams.length > 0) {
    const paramsName = paramsNames.get(operation)
    if (!paramsName) {
      throw new Error(
        `typespec-go: no params struct name resolved for ${operation.operation.name}`,
      )
    }
    parameters.push({
      name: 'params',
      type: paramsName,
    })
  }

  return parameters
}

function pathCode(operation: GoOperation): ay.Children {
  const errPrefix =
    operation.response || isTextResponse(operation) ? 'nil, ' : ''
  const guards: ay.Children[] = operation.pathParams.map(
    (parameter) =>
      ay.code`if ${localName(parameter.name)} == "" {
          return ${errPrefix}${go.std.fmt.Errorf}("openmeter: %s must not be empty: %w", ${JSON.stringify(localName(parameter.name))}, ErrEmptyID)
        }`,
  )
  const substitutions: ay.Children[] = operation.pathParams.map(
    (parameter) =>
      ay.code`path = replacePathParam(path, ${JSON.stringify(parameter.name)}, ${localName(parameter.name)})`,
  )

  return (
    <ay.List joiner={'\n\n'}>
      {[
        ...guards,
        ay.code`path := ${JSON.stringify(operation.path)}`,
        ...substitutions,
      ]}
    </ay.List>
  )
}

function serviceReceiver(serviceRefkey: Refkey) {
  return (
    <go.FunctionReceiver
      name="s"
      type={
        <go.Pointer>
          <go.Reference refkey={serviceRefkey} />
        </go.Pointer>
      }
    />
  )
}

function Seq2Type({ element }: { element: string }) {
  return (
    <>
      {iter.Seq2}[{element}, error]
    </>
  )
}

function httpMethod(verb: string): ay.Children {
  switch (verb.toUpperCase()) {
    case 'GET':
      return go.std.net.http.MethodGet
    case 'POST':
      return go.std.net.http.MethodPost
    case 'PUT':
      return go.std.net.http.MethodPut
    case 'PATCH':
      return go.std.net.http.MethodPatch
    case 'DELETE':
      return go.std.net.http.MethodDelete
    default:
      return JSON.stringify(verb.toUpperCase())
  }
}

function isTextResponse(operation: GoOperation): boolean {
  return operation.responseContentType?.startsWith('text/') ?? false
}

/**
 * Resolves the params struct name for every operation that has query
 * parameters.
 *
 * Params structs are preferably named after the page element
 * (CustomerListParams). Distinct operations can legitimately page over the
 * same element with different query shapes (list_prices supports sort,
 * list_overrides does not); naming both after the element would merge two
 * different shapes into whichever struct happens to be discovered first.
 * Every operation in such a conflicted group therefore gets its
 * operation-derived name instead, keeping the outcome independent of
 * operation iteration order.
 */
export function resolveListParamsNames(
  program: Program,
  operations: GoOperation[],
): Map<GoOperation, string> {
  const groups = new Map<string, GoOperation[]>()
  for (const operation of operations) {
    if (operation.queryParams.length === 0) {
      continue
    }
    const name = listParamsName(program, operation)
    const group = groups.get(name)
    if (group) {
      group.push(operation)
    } else {
      groups.set(name, [operation])
    }
  }

  const resolved = new Map<GoOperation, string>()
  const shapes = new Map<string, string>()
  for (const [name, group] of groups) {
    const signatures = new Set(
      group.map((operation) => querySignature(program, operation.queryParams)),
    )
    for (const operation of group) {
      const finalName =
        signatures.size > 1 ? operationParamsName(program, operation) : name
      const signature = querySignature(program, operation.queryParams)
      const existing = shapes.get(finalName)
      if (existing !== undefined && existing !== signature) {
        throw new Error(
          `typespec-go: params struct ${finalName} would be emitted with two different query shapes; add a distinct @friendlyName or @operationId to one of the operations`,
        )
      }
      shapes.set(finalName, signature)
      resolved.set(operation, finalName)
    }
  }

  return resolved
}

/**
 * Canonical signature of an operation's query parameter list mirroring
 * exactly what the generated params struct and its values() body depend on:
 * parameter order, wire names, codec kinds, optionality, and rendered Go
 * field types. Two operations may share a params struct only when their
 * signatures are equal.
 */
function querySignature(program: Program, parameters: GoParameter[]): string {
  return parameters
    .map((parameter) => {
      const codec = parameter.queryCodec
      switch (codec?.kind) {
        case 'page':
          return 'page'
        case 'cursorPage':
          return 'cursorPage'
        case 'sort':
          return `sort:${parameter.name}`
        case 'deepObject':
          return `deepObject:${parameter.name}:{${[
            ...codec.model.properties.values(),
          ]
            .map(
              (property) =>
                `${property.name}:${resolveEncodedName(program, property, 'application/json')}:${filterFieldSignature(program, property.type)}`,
            )
            .join(',')}}`
        case 'array': {
          if (
            parameter.type.kind !== 'Model' ||
            !parameter.type.indexer?.value
          ) {
            throw new Error(
              `array query parameter ${parameter.name} has no element type`,
            )
          }
          return `array:${parameter.name}:${codec.explode}:${scalarSignature(program, parameter.type.indexer.value)}`
        }
        default:
          return `scalar:${parameter.name}:${parameter.property.optional}:${scalarSignature(program, parameter.type)}`
      }
    })
    .join(';')
}

function filterFieldSignature(program: Program, type: Type): string {
  const kind = queryFilterKind(program, type)
  return kind === 'scalar' ? `scalar:${scalarSignature(program, type)}` : kind
}

function scalarSignature(program: Program, type: Type): string {
  const mapped = goType(program, type).type
  return typeof mapped === 'string' ? mapped : queryScalarKind(program, type)
}

function operationParamsName(program: Program, operation: GoOperation): string {
  return `${goExportedName(operationBaseName(program, operation.operation))}Params`
}

function listParamsName(program: Program, operation: GoOperation): string {
  if (operation.pagination && operation.response) {
    try {
      return `${pageElement(program, operation.response)}ListParams`
    } catch {
      // A cursor-bearing operation can return a non-standard envelope. Keep a
      // collision-free operation name rather than guessing an element type.
    }
  }

  return operationParamsName(program, operation)
}

/**
 * All-iterators surface only the page elements, so they are emitted only for
 * the canonical {data, meta} page envelope. Any extra response field (for
 * example GovernanceQueryResponse.errors, which reports partial failures)
 * would be silently dropped from the iteration; such operations only get the
 * plain method that returns the full envelope.
 */
export function isPlainPageEnvelope(
  program: Program,
  response: Type | undefined,
): boolean {
  if (response?.kind !== 'Model') {
    return false
  }

  const properties = [...walkPropertiesInherited(response)].filter(
    (property) =>
      !isStatusCode(program, property) && !isHeader(program, property),
  )

  return (
    properties.length === 2 &&
    properties.some(
      (property) =>
        property.name === 'data' &&
        property.type.kind === 'Model' &&
        $(program).array.is(property.type),
    ) &&
    properties.some((property) => property.name === 'meta')
  )
}

function allIteratorDoc(operation: GoOperation, element: string): string {
  const listName = goExportedName(operation.methodName)
  return `${listName}All returns an iterator over all ${element} results, fetching pages of ${listName} transparently. Iteration stops at the first error, which is yielded as the second value.`
}

function pageElement(program: Program, response: Type): string {
  if (response.kind !== 'Model') {
    throw new Error('paginated response must be a model')
  }
  const data = response.properties.get('data')
  if (!data || data.type.kind !== 'Model' || !$(program).array.is(data.type)) {
    throw new Error(
      `${typeName(program, response)} must contain an array data property`,
    )
  }
  const element = data.type.indexer?.value
  if (!element) {
    throw new Error(`${typeName(program, response)} data array has no element`)
  }

  return readReferenceName(program, element)
}

function readReferenceName(program: Program, type: Type): string {
  const mapped = goType(program, type).type
  return typeof mapped === 'string' ? mapped : typeName(program, type)
}

// Locals every generated method body may declare (including the receiver and
// the paginate callback arguments that capture path parameters); a path
// parameter with one of these names would shadow them and emit non-compiling
// or subtly wrong Go.
const reservedMethodLocals = new Set([
  'after',
  'before',
  'ctx',
  'err',
  'out',
  'page',
  'pageParams',
  'params',
  'path',
  'q',
  'req',
  'request',
  'resp',
  's',
  'size',
])

export function localName(name: string): string {
  const exported = goExportedName(name)
  const acronym = exported.match(/^([A-Z]{2,})([A-Z][a-z].*)$/)
  const local = /^[A-Z0-9]+$/.test(exported)
    ? exported.toLowerCase()
    : acronym?.[1] && acronym[2]
      ? acronym[1].toLowerCase() + acronym[2]
      : exported.charAt(0).toLowerCase() + exported.slice(1)

  return reservedMethodLocals.has(local) ? `${local}Param` : local
}
