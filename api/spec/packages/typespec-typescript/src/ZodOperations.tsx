import { type Children, For, refkey } from '@alloy-js/core'
import {
  ObjectExpression,
  ObjectProperty,
  VarDeclaration,
} from '@alloy-js/typescript'
import {
  getFriendlyName,
  type ModelProperty,
  type Operation,
  type Program,
  type Type,
} from '@typespec/compiler'
import { $ } from '@typespec/compiler/typekit'
import { getAllHttpServices } from '@typespec/http'
import { getExtensions, getOperationId } from '@typespec/openapi'
import { getQueryCodec, type QueryCodec } from '@openmeter/typespec-sdk'
import { ZodSchema } from './components/ZodSchema.jsx'
import { isSuccessStatus } from './http-status.js'
import {
  callPart,
  CoerceContext,
  toCamelCase,
  useWireMode,
  zodMemberExpr,
} from './utils.jsx'

/**
 * Per-operation request and response Zod schemas.
 *
 * The component-level walk emits one schema per data type. This walk adds the
 * HTTP layer on top: for every operation it emits the assembled query- and
 * path-parameter objects (with URL-appropriate coercion), the request body, and
 * the success response body. Bodies reference the shared component schemas via
 * refkey so the reuse the component emitter buys is preserved; only the param
 * objects, which have no standalone component, are built inline.
 */

/** Refkey families so per-operation schemas can be cross-referenced if needed. */
const pathParamsSym = Symbol.for('typespec-typescript.op.pathParams')
const queryParamsSym = Symbol.for('typespec-typescript.op.queryParams')
const bodySym = Symbol.for('typespec-typescript.op.body')
const responseSym = Symbol.for('typespec-typescript.op.response')

export interface OperationSchema {
  /**
   * Declared name before prefix stripping (e.g. `CreateCustomerBody`). Joined
   * into the emitter's name pool so strips never collide with these.
   */
  baseName: string
  /** Render the declaration under its final (post-strip) name. */
  render: (name: string) => Children
}

// Customer-visibility markers from the spec's shared/consts.tsp: x-private is
// "private and should not be exposed to customers", x-internal is "internal and
// should not be used by customers". Both are emitted but quarantined under the
// `client.internal.*` sub-client so the audience split stays visible at every
// call site — the SDK is also how internal consumers call the API, so dropping
// x-private operations would just push those callers back to hand-rolled HTTP.
// x-unstable operations stay in the public surface: most of the young v3 API
// carries that marker, and it flags maturity, not audience.
function hasExtension(
  program: Program,
  op: Operation,
  key: `x-${string}`,
): boolean {
  // The walked operation is an `extends`/`op is` instance; the @extension
  // decorators may live on it or on the source operation it was cloned from,
  // so the whole source chain is consulted.
  for (
    let current: Operation | undefined = op;
    current;
    current = current.sourceOperation
  ) {
    if (getExtensions(program, current).get(key) === true) {
      return true
    }
  }
  return false
}

/**
 * Whether an operation belongs to the `client.internal.*` surface. Internal
 * operations are emitted like any other (funcs, envelope types, zod schemas),
 * but their grouped-client methods live under `client.internal.*` instead of
 * the public sub-clients (see emitter.tsx). x-private implies the internal
 * surface too: it marks a stricter audience than x-internal, so it must never
 * surface publicly, but internal consumers still call it through the SDK.
 */
export function isInternalOperation(program: Program, op: Operation): boolean {
  return (
    hasExtension(program, op, 'x-internal') ||
    hasExtension(program, op, 'x-private')
  )
}

/**
 * Collect every HTTP operation in the program, de-duplicated by the underlying
 * TypeSpec `Operation` (the same operation surfaces under multiple service
 * namespaces — OpenMeter and MeteringAndBilling — and must not be emitted
 * twice). Operations marked x-internal or x-private are collected like any
 * other and later routed to the `client.internal.*` surface.
 */
export function collectHttpOperations(
  program: Program,
  includeServices?: string[],
): Operation[] {
  const [services] = getAllHttpServices(program)
  const included =
    includeServices && includeServices.length > 0
      ? services.filter((s) => includeServices.includes(s.namespace.name))
      : services
  // The same logical endpoint is declared under multiple service namespaces
  // (OpenMeter and MeteringAndBilling) as distinct `extends` instances, and
  // `@sharedRoute` content-type variants share one id too. De-dupe by the
  // stable operation id and keep the first occurrence so each endpoint emits a
  // single set of schemas.
  const seen = new Set<string>()
  const result: Operation[] = []
  for (const service of included) {
    for (const httpOp of service.operations) {
      const id = operationBaseName(program, httpOp.operation)
      if (seen.has(id)) {
        continue
      }
      seen.add(id)
      result.push(httpOp.operation)
    }
  }
  return result
}

/**
 * `create-customer` / `Foo_bar` -> `CreateCustomer` / `FooBar`.
 *
 * An operation-level `@friendlyName` takes precedence over the operation id:
 * `@sharedRoute` content-type variants share one operation id (one OpenAPI
 * operation), and a friendly name is how the spec marks a variant that must
 * surface as its own SDK operation (e.g. `queryMeterCsv`) instead of being
 * collapsed into its JSON sibling.
 */
export function operationBaseName(program: Program, op: Operation): string {
  const id =
    getFriendlyName(program, op) || getOperationId(program, op) || op.name
  return id
    .split(/[-_/\s]+/)
    .filter(Boolean)
    .map((part: string) => part.charAt(0).toUpperCase() + part.slice(1))
    .join('')
}

interface ParamLeaf {
  name: string
  prop: ModelProperty
  codec?: QueryCodec
}

function paramObject(
  params: ParamLeaf[],
  camelizeInCamelPass: boolean,
): Children {
  // `p.name` is the HTTP-binding wire name. Camelize it only for the public
  // (camel) pass; the wire pass (emitted under WireModeContext, see
  // emitter.tsx's `…Wire` re-render) must keep the raw wire name so a
  // `*QueryParamsWire` schema actually matches the querystring sent on the
  // wire — otherwise it silently describes the wrong (camelCase) shape.
  const wire = useWireMode()
  const camelize = camelizeInCamelPass && !wire
  return (
    <CoerceContext.Provider value={true}>
      {zodMemberExpr(
        callPart(
          'object',
          <ObjectExpression>
            <For each={params} comma hardline enderPunctuation>
              {(p) => (
                <ObjectProperty name={camelize ? toCamelCase(p.name) : p.name}>
                  {wire && p.codec ? (
                    <CoerceContext.Provider value={false}>
                      <ZodSchema
                        type={p.prop}
                        valueType={p.codec.wireType}
                        nested
                      />
                    </CoerceContext.Provider>
                  ) : (
                    <ZodSchema type={p.prop} nested />
                  )}
                </ObjectProperty>
              )}
            </For>
          </ObjectExpression>,
        ),
      )}
    </CoerceContext.Provider>
  )
}

/**
 * Build the request/response schema declarations for a single operation.
 * Returns the renderable declarations along with the names they declare so the
 * caller can run them through prefix-stripping/name-policy resolution.
 */
export function operationSchemas(
  program: Program,
  op: Operation,
  bodyOverrides: Map<string, Type>,
): OperationSchema[] {
  const tk = $(program)
  const httpOp = tk.httpOperation.get(op)
  const base = operationBaseName(program, op)
  const out: OperationSchema[] = []

  const pathParams: ParamLeaf[] = []
  const queryParams: ParamLeaf[] = []
  for (const param of httpOp.parameters.parameters) {
    if (param.type === 'path') {
      pathParams.push({ name: param.name, prop: param.param })
    } else if (param.type === 'query') {
      queryParams.push({
        name: param.name,
        prop: param.param,
        codec: getQueryCodec(program, param.param),
      })
    }
    // headers are transport metadata; intentionally skipped.
  }

  if (pathParams.length > 0) {
    out.push({
      baseName: `${base}PathParams`,
      render: (name) => (
        <VarDeclaration export name={name} refkey={refkey(op, pathParamsSym)}>
          {paramObject(pathParams, false)}
        </VarDeclaration>
      ),
    })
  }

  if (queryParams.length > 0) {
    out.push({
      baseName: `${base}QueryParams`,
      render: (name) => (
        <VarDeclaration export name={name} refkey={refkey(op, queryParamsSym)}>
          {paramObject(queryParams, true)}
        </VarDeclaration>
      ),
    })
  }

  // The body the func actually sends: a shared-route JSON override (e.g. the
  // single-or-batch ingest union, or a response-only variant like queryMeterCsv
  // whose declared op omits the body) when present, else the op's own body. The
  // boundary mapper walks this schema, so it must match the wire shape.
  const bodyType = bodyOverrides.get(base) ?? httpOp.parameters.body?.type
  if (bodyType) {
    out.push({
      baseName: `${base}Body`,
      render: (name) => (
        <VarDeclaration export name={name} refkey={refkey(op, bodySym)}>
          {bodySchemaRef(program, bodyType)}
        </VarDeclaration>
      ),
    })
  }

  const responseType = successBodyType(program, httpOp)
  if (responseType) {
    out.push({
      baseName: `${base}Response`,
      render: (name) => (
        <VarDeclaration export name={name} refkey={refkey(op, responseSym)}>
          {bodySchemaRef(program, responseType)}
        </VarDeclaration>
      ),
    })
  }

  return out
}

/**
 * Reference the component schema for a body type, or inline it when it is an
 * anonymous/non-declaration type with no standalone component.
 */
function bodySchemaRef(_program: Program, type: Type): Children {
  // `nested` ZodSchema emits a reference (refkey member expression) for
  // declaration types and inlines anonymous ones, matching the component walk.
  return <ZodSchema type={type} nested />
}

/**
 * The body type of the first 2xx response that carries one. Error responses are
 * available as their own component schemas (e.g. `badRequest`); the per-op
 * `Response` schema models the success payload a caller validates.
 */
function successBodyType(
  program: Program,
  httpOp: ReturnType<ReturnType<typeof $>['httpOperation']['get']>,
): Type | undefined {
  for (const response of httpOp.responses) {
    if (!isSuccessStatus(response.statusCodes)) {
      continue
    }
    for (const content of response.responses) {
      if (content.body) {
        return content.body.type
      }
    }
  }
  return undefined
}
