import { type Children, refkey } from '@alloy-js/core'
import { MemberExpression } from '@alloy-js/typescript'
import type { Type } from '@typespec/compiler'
import { useTsp } from '@typespec/emitter-framework'
import { activeRefkeySym, shouldReference, useWireMode } from '../utils.jsx'
import { zodBaseSchemaParts } from '../zodBaseSchema.jsx'
import { zodConstraintsParts } from '../zodConstraintsParts.jsx'
import { zodDescriptionParts } from '../zodDescriptionParts.jsx'
import { zodMemberParts } from '../zodMemberParts.jsx'
import { ZodCustomTypeComponent } from './ZodCustomTypeComponent.jsx'

export interface ZodSchemaProps {
  readonly type: Type
  /** Overrides the property's value type while preserving its member metadata. */
  readonly valueType?: Type
  readonly nested?: boolean
}

/**
 * Component that translates a TypeSpec type into the Zod type.
 */
export function ZodSchema(props: ZodSchemaProps): Children {
  const { $ } = useTsp()
  const rkSym = activeRefkeySym(useWireMode())

  if (!props.nested) {
    return (
      <MemberExpression>
        {zodBaseSchemaParts(props.type)}
        {zodConstraintsParts(props.type)}
        {zodDescriptionParts(props.type)}
      </MemberExpression>
    )
  }

  const { member, type } = $.modelProperty.is(props.type)
    ? { member: props.type, type: props.type.type }
    : { type: props.type, member: undefined }
  const valueType = props.valueType ?? type

  if (shouldReference($.program, valueType)) {
    return (
      <ZodCustomTypeComponent type={valueType} member={member} reference>
        <MemberExpression>
          <MemberExpression.Part refkey={refkey(valueType, rkSym)} />
          {zodMemberParts(member)}
        </MemberExpression>
      </ZodCustomTypeComponent>
    )
  }

  return (
    <ZodCustomTypeComponent type={valueType} member={member} reference>
      <MemberExpression>
        {zodBaseSchemaParts(valueType)}
        {zodConstraintsParts(valueType, member)}
        {zodMemberParts(member)}
        {zodDescriptionParts(valueType, member)}
      </MemberExpression>
    </ZodCustomTypeComponent>
  )
}
