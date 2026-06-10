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
import { getOperationId } from '@typespec/openapi'
import { ZodSchema } from './components/ZodSchema.jsx'
import { callPart, CoerceContext, zodMemberExpr } from './utils.jsx'

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

/**
 * Collect every HTTP operation in the program, de-duplicated by the underlying
 * TypeSpec `Operation` (the same operation surfaces under multiple service
 * namespaces — OpenMeter and MeteringAndBilling — and must not be emitted
 * twice).
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
}

function paramObject(params: ParamLeaf[]): Children {
  return (
    <CoerceContext.Provider value={true}>
      {zodMemberExpr(
        callPart(
          'object',
          <ObjectExpression>
            <For each={params} comma hardline enderPunctuation>
              {(p) => (
                <ObjectProperty name={p.name}>
                  <ZodSchema type={p.prop} nested />
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
      queryParams.push({ name: param.name, prop: param.param })
    }
    // headers are transport metadata; intentionally skipped.
  }

  if (pathParams.length > 0) {
    out.push({
      baseName: `${base}PathParams`,
      render: (name) => (
        <VarDeclaration export name={name} refkey={refkey(op, pathParamsSym)}>
          {paramObject(pathParams)}
        </VarDeclaration>
      ),
    })
  }

  if (queryParams.length > 0) {
    out.push({
      baseName: `${base}QueryParams`,
      render: (name) => (
        <VarDeclaration export name={name} refkey={refkey(op, queryParamsSym)}>
          {paramObject(queryParams)}
        </VarDeclaration>
      ),
    })
  }

  const bodyType = httpOp.parameters.body?.type
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
