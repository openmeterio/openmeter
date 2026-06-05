import type { ModelProperty } from '@typespec/compiler'
import type { Typekit } from '@typespec/compiler/typekit'
import { useTsp } from '@typespec/emitter-framework'
import { ValueExpression } from '@typespec/emitter-framework/typescript'
import { callPart } from './utils.jsx'
import { usesBigIntBase } from './zodBaseSchema.jsx'

export function zodMemberParts(member?: ModelProperty) {
  const { $ } = useTsp()
  return [...optionalParts($, member), ...defaultParts($, member)]
}

function defaultParts($: Typekit, member?: ModelProperty) {
  if (!member?.defaultValue) {
    return []
  }

  // Integer defaults need an explicit literal: the base schema is either
  // `z.coerce.bigint()` (int64/uint64 → `1n`) or `z.number().int()` (every
  // other integer → `1`). TypeSpec's arbitrary-precision `integer` would
  // otherwise render as a `bigint` literal via ValueExpression and fail to
  // typecheck against `z.number()`.
  if (
    member.defaultValue.valueKind === 'NumericValue' &&
    $.scalar.is(member.type) &&
    $.scalar.extendsInteger(member.type)
  ) {
    if (usesBigIntBase($, member.type)) {
      const big = member.defaultValue.value.asBigInt()
      if (big !== null) {
        return [callPart('default', `${big}n`)]
      }
    } else {
      const num = member.defaultValue.value.asNumber()
      if (num !== null) {
        return [callPart('default', `${num}`)]
      }
    }
  }

  return [
    callPart('default', [<ValueExpression value={member.defaultValue} />]),
  ]
}

function optionalParts(_$: Typekit, member?: ModelProperty) {
  if (!member?.optional) {
    return []
  }

  return [callPart('optional')]
}
