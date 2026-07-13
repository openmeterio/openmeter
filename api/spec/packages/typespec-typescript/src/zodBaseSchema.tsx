import { type Children, For, refkey } from '@alloy-js/core'
import {
  ArrayExpression,
  MemberExpression,
  ObjectExpression,
  ObjectProperty,
} from '@alloy-js/typescript'
import type {
  Enum,
  LiteralType,
  Model,
  Scalar,
  Tuple,
  Type,
  Union,
} from '@typespec/compiler'
import type { Typekit } from '@typespec/compiler/typekit'
import { useTsp } from '@typespec/emitter-framework'
import { ZodCustomTypeComponent } from './components/ZodCustomTypeComponent.jsx'
import { ZodSchema } from './components/ZodSchema.jsx'
import { reportDiagnostic } from './lib.js'
import {
  activeRefkeySym,
  bodyProperties,
  callPart,
  emitsAsIntersection,
  idPart,
  isDeclaration,
  isRecord,
  schemaPropertyName,
  shouldReference,
  subtreeReachesType,
  toCamelCase,
  useCoerce,
  useDeclaringType,
  useWireMode,
  zodMemberExpr,
} from './utils.jsx'

/**
 * Leading `z.coerce.<method>` parts when emitting in a coerced context (query/
 * path params), or plain `z.<method>` otherwise. Coercion only makes sense for
 * the string/number/boolean primitives.
 */
function coercedPrimitive(method: string, ...rest: Children[]) {
  if (useCoerce()) {
    return zodMemberExpr(idPart('coerce'), callPart(method), ...rest)
  }
  return zodMemberExpr(callPart(method), ...rest)
}

/**
 * Returns the identifier parts for the base Zod schema for a given TypeSpec type.
 */
export function zodBaseSchemaParts(type: Type) {
  const { $ } = useTsp()

  switch (type.kind) {
    case 'Intrinsic':
      return intrinsicBaseType(type)
    case 'String':
    case 'Number':
    case 'Boolean':
      return literalBaseType(type)
    case 'Scalar':
      return scalarBaseType($, type)
    case 'Model':
      return modelBaseType(type)
    case 'Union':
      return unionBaseType(type)
    case 'Enum':
      return enumBaseType(type)
    case 'ModelProperty':
      return zodBaseSchemaParts(type.type)
    case 'EnumMember':
      return type.value
        ? literalBaseType($.literal.create(type.value))
        : literalBaseType($.literal.create(type.name))
    case 'Tuple':
      return tupleBaseType(type)
    default:
      reportDiagnostic($.program, {
        code: 'unsupported-type',
        target: type,
        format: { kind: type.kind },
      })
      return zodMemberExpr(callPart('any'))
  }
}

/**
 * Returns true when {@link scalarBaseType} would emit `z.coerce.bigint()` for
 * this scalar. Reserved for the 64-bit integer scalars whose range exceeds
 * `Number.MAX_SAFE_INTEGER`; every other integer (plain `integer`, int32 and
 * narrower, safeint) stays a JS `number` so the inferred type is not virally
 * `bigint` for consumers. Kept colocated with the base-type decision so the
 * constraint emitter renders literal bounds as bigints (`1n`) rather than
 * numbers (`1`) for matching types.
 */
export function usesBigIntBase($: Typekit, type: Scalar): boolean {
  // extendsInt64/extendsUint64 use value-range assignability, so int8/16/32 and
  // safeint (all within Number.MAX_SAFE_INTEGER) also satisfy them. Exclude those
  // so only true 64-bit scalars coerce to bigint.
  const isInt64 =
    $.scalar.extendsInt64(type) &&
    !$.scalar.extendsInt32(type) &&
    !$.scalar.extendsSafeint(type)
  const isUint64 =
    $.scalar.extendsUint64(type) &&
    !$.scalar.extendsUint32(type) &&
    !$.scalar.extendsSafeint(type)
  return isInt64 || isUint64
}

function literalBaseType(type: LiteralType) {
  switch (type.kind) {
    case 'String':
      return zodMemberExpr(callPart('literal', JSON.stringify(type.value)))
    case 'Number':
    case 'Boolean':
      return zodMemberExpr(callPart('literal', `${type.value}`))
  }
}

function scalarBaseType($: Typekit, type: Scalar) {
  // Captured before the extendsX predicate chain: each failed predicate
  // narrows `type`, so by the fallback arms the compiler no longer lets the
  // Scalar's own members through.
  const scalarName = type.name
  if (type.baseScalar && shouldReference($.program, type.baseScalar)) {
    return (
      <MemberExpression.Part
        refkey={refkey(type.baseScalar, activeRefkeySym(useWireMode()))}
      />
    )
  }

  if ($.scalar.extendsBoolean(type)) {
    return coercedPrimitive('boolean')
  }

  if ($.scalar.extendsNumeric(type)) {
    if ($.scalar.extendsInteger(type)) {
      if (usesBigIntBase($, type)) {
        // JSON numbers parse as JS `number`; coerce so int64/uint64 fields
        // accept them instead of requiring a `bigint` literal at the call site.
        return zodMemberExpr(idPart('coerce'), callPart('bigint'))
      }
      return coercedPrimitive('number', callPart('int'))
    }
    // floats and such; lacking a decimal type this is the best we can do.
    return coercedPrimitive('number')
  }

  if ($.scalar.extendsString(type)) {
    // `url` is treated as a plain string: Zod 4 deprecated the chained
    // `z.string().url()` in favor of a top-level `z.url()`, and URL-format
    // validation is not worth diverging the base type for here.
    return coercedPrimitive('string')
  }

  if ($.scalar.extendsBytes(type)) {
    reportDiagnostic($.program, {
      code: 'unsupported-type',
      target: type,
      format: { kind: `bytes scalar '${scalarName}'` },
    })
    return zodMemberExpr(callPart('any'))
  }

  if ($.scalar.extendsPlainDate(type)) {
    return zodMemberExpr(idPart('coerce'), callPart('date'))
  }

  if ($.scalar.extendsPlainTime(type)) {
    return zodMemberExpr(callPart('string'), callPart('time'))
  }

  if ($.scalar.extendsUtcDateTime(type)) {
    const encoding = $.scalar.getEncoding(type)
    if (encoding?.encoding === 'unixTimestamp') {
      return scalarBaseType($, encoding.type)
    }
    if (encoding === undefined || encoding.encoding === 'rfc3339') {
      return dateTimeBaseType()
    }
    return scalarBaseType($, encoding.type)
  }

  if ($.scalar.extendsOffsetDateTime(type)) {
    const encoding = $.scalar.getEncoding(type)
    if (encoding === undefined || encoding.encoding === 'rfc3339') {
      return dateTimeBaseType()
    }
    return scalarBaseType($, encoding.type)
  }

  if ($.scalar.extendsDuration(type)) {
    const encoding = $.scalar.getEncoding(type)
    if (encoding === undefined || encoding.encoding === 'ISO8601') {
      return zodMemberExpr(callPart('string'), callPart('duration'))
    }
    return scalarBaseType($, encoding.type)
  }

  reportDiagnostic($.program, {
    code: 'unsupported-type',
    target: type,
    format: { kind: `scalar '${scalarName}'` },
  })
  return zodMemberExpr(callPart('any'))
}

/**
 * RFC 3339 date-time scalars diverge between the two schema passes: the public
 * pass types them `z.date()` (the SDK surface takes and returns `Date`; the
 * runtime wire mapper converts at the request/response boundary), while the
 * wire pass keeps the RFC 3339 string the JSON payload actually carries.
 */
function dateTimeBaseType() {
  if (useWireMode()) {
    return zodMemberExpr(callPart('string'), callPart('datetime'))
  }
  return zodMemberExpr(callPart('date'))
}

function enumBaseType(type: Enum) {
  return zodMemberExpr(
    callPart(
      'enum',
      <ArrayExpression>
        <For each={type.members.values()} comma line>
          {(member) => (
            <ZodCustomTypeComponent
              type={member}
              Declaration={(props: { children?: Children }) => props.children}
              declarationProps={{}}
              declare
            >
              {JSON.stringify(member.value ?? member.name)}
            </ZodCustomTypeComponent>
          )}
        </For>
      </ArrayExpression>,
    ),
  )
}

function tupleBaseType(type: Tuple) {
  return zodMemberExpr(
    callPart(
      'tuple',
      <ArrayExpression>
        <For each={type.values} comma line>
          {(item) => <ZodSchema type={item} nested />}
        </For>
      </ArrayExpression>,
    ),
  )
}

function modelBaseType(type: Model) {
  const { $ } = useTsp()
  const wire = useWireMode()
  const rkSym = activeRefkeySym(wire)
  // Closed objects are strict in the wire pass so a leaked-camelCase or unknown
  // wire key is rejected. Open models (record spread, `emitsAsIntersection`) stay
  // permissive — strict would defeat the record arm that exists to accept them.
  const objectCall =
    wire && !emitsAsIntersection($.program, type) ? 'strictObject' : 'object'

  if ($.array.is(type)) {
    return zodMemberExpr(
      callPart('array', <ZodSchema type={type.indexer?.value} nested />),
    )
  }

  let recordPart: Children
  if (
    isRecord($.program, type) ||
    (!!type.baseModel &&
      isRecord($.program, type.baseModel) &&
      !isDeclaration($.program, type.baseModel))
  ) {
    const indexer = (type.indexer ?? type.baseModel?.indexer)!
    recordPart = zodMemberExpr(
      callPart(
        'record',
        <ZodSchema type={indexer.key} nested />,
        <ZodSchema type={indexer.value} nested />,
      ),
    )
  }

  const declaringType = useDeclaringType()
  let memberPart: Children
  // Drop HTTP envelope metadata (`@statusCode`, `@header`) so response/error
  // models emit only their JSON body shape instead of bogus fields like
  // `_: z.literal(200)`.
  const props = bodyProperties($.program, type)
  if (props.length > 0) {
    const members = (
      <ObjectExpression>
        <For each={props} comma hardline enderPunctuation>
          {(prop) => {
            const key = schemaPropertyName($.program, prop, wire)
            const isCyclic =
              declaringType !== undefined &&
              subtreeReachesType($.program, prop.type, declaringType)
            const propertyContent = isCyclic ? (
              <>
                get {key}() {'{ return '}
                <ZodSchema type={prop} nested />
                {'; }'}
              </>
            ) : (
              <ObjectProperty name={key}>
                <ZodSchema type={prop} nested />
              </ObjectProperty>
            )
            return (
              <ZodCustomTypeComponent
                type={prop}
                declare
                Declaration={ObjectProperty}
                declarationProps={{ name: key }}
              >
                {propertyContent}
              </ZodCustomTypeComponent>
            )
          }}
        </For>
      </ObjectExpression>
    )
    memberPart = zodMemberExpr(callPart(objectCall, members))
  }

  const hasReferencedBase =
    !!type.baseModel && shouldReference($.program, type.baseModel)

  let parts: Children

  if (!memberPart && !recordPart) {
    // A subclass that adds nothing of its own (e.g. an error whose only
    // property was the stripped `@statusCode`) is just its base schema; avoid
    // an empty `base.merge(z.object({}))`.
    if (hasReferencedBase) {
      return <MemberExpression.Part refkey={refkey(type.baseModel!, rkSym)} />
    }
    parts = zodMemberExpr(callPart(objectCall, <ObjectExpression />))
  } else if (memberPart && recordPart) {
    parts = zodMemberExpr(callPart('intersection', memberPart, recordPart))
  } else {
    parts = memberPart ?? recordPart
  }

  if (hasReferencedBase) {
    // `.merge()` only exists on `ZodObject`. When the base is emitted as a
    // `z.intersection(...)` (members + record spread), compose with
    // `z.intersection` instead so the subclass schema stays valid.
    if (emitsAsIntersection($.program, type.baseModel!)) {
      const baseRef = (
        <MemberExpression>
          <MemberExpression.Part refkey={refkey(type.baseModel!, rkSym)} />
        </MemberExpression>
      )
      return zodMemberExpr(callPart('intersection', baseRef, parts))
    }
    return (
      <MemberExpression>
        <MemberExpression.Part refkey={refkey(type.baseModel!, rkSym)} />
        <MemberExpression.Part id="merge" />
        <MemberExpression.Part args={[parts]} />
      </MemberExpression>
    )
  }

  return parts
}

function unionBaseType(type: Union) {
  const { $ } = useTsp()

  const discriminated = $.union.getDiscriminatedUnion(type)

  if ($.union.isExpression(type) || !discriminated) {
    return zodMemberExpr(
      callPart(
        'union',
        <ArrayExpression>
          <For each={type.variants} comma line>
            {(_name, variant) => <ZodSchema type={variant.type} nested />}
          </For>
        </ArrayExpression>,
      ),
    )
  }

  // The discriminator key tracks the variant member keys: camelized in the public
  // pass, raw wire name in the wire pass, so `z.discriminatedUnion`'s key matches
  // the emitted variant shapes. The discriminator value (`variant.name`) is a
  // literal and is never renamed.
  const wire = useWireMode()
  const propKey = wire
    ? discriminated.options.discriminatorPropertyName
    : toCamelCase(discriminated.options.discriminatorPropertyName)
  const envKey = wire
    ? discriminated.options.envelopePropertyName
    : toCamelCase(discriminated.options.envelopePropertyName)
  const unionArgs = [
    `"${propKey}"`,
    <ArrayExpression>
      <For each={Array.from(type.variants.values())} comma line>
        {(variant) => {
          if (discriminated.options.envelope === 'object') {
            const envelope = $.model.create({
              properties: {
                [propKey]: $.modelProperty.create({
                  name: propKey,
                  type: $.literal.create(variant.name as string),
                }),
                [envKey]: $.modelProperty.create({
                  name: envKey,
                  type: variant.type,
                }),
              },
            })
            return <ZodSchema type={envelope} nested />
          }
          return <ZodSchema type={variant.type} nested />
        }}
      </For>
    </ArrayExpression>,
  ]

  return zodMemberExpr(callPart('discriminatedUnion', ...unionArgs))
}

function intrinsicBaseType(type: Type) {
  if (type.kind === 'Intrinsic') {
    switch (type.name) {
      case 'null':
        return zodMemberExpr(callPart('null'))
      case 'never':
        return zodMemberExpr(callPart('never'))
      case 'unknown':
        return zodMemberExpr(callPart('unknown'))
      case 'void':
        return zodMemberExpr(callPart('void'))
      default:
        return zodMemberExpr(callPart('any'))
    }
  }
  return zodMemberExpr(callPart('any'))
}
