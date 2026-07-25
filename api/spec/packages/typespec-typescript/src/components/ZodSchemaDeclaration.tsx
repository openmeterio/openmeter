import * as ay from '@alloy-js/core'
import * as ts from '@alloy-js/typescript'
import { getFriendlyName } from '@typespec/compiler'
import { useTsp } from '@typespec/emitter-framework'
import {
  activeRefkeySym,
  DeclaringTypeContext,
  useWireMode,
} from '../utils.jsx'
import { ZodCustomTypeComponent } from './ZodCustomTypeComponent.jsx'
import { ZodSchema, type ZodSchemaProps } from './ZodSchema.jsx'

interface ZodSchemaDeclarationProps
  extends
    Omit<ts.VarDeclarationProps, 'type' | 'name' | 'value' | 'kind'>,
    ZodSchemaProps {
  readonly name?: string
}

/**
 * Declare a Zod schema.
 */
export function ZodSchemaDeclaration(props: ZodSchemaDeclarationProps) {
  const { $ } = useTsp()
  const internalRk = ay.refkey(props.type, activeRefkeySym(useWireMode()))
  const [zodSchemaProps, varDeclProps] = ay.splitProps(props, [
    'type',
    'nested',
  ]) as [ZodSchemaDeclarationProps, ts.VarDeclarationProps]

  const refkeys = [props.refkey ?? []].flat()
  refkeys.push(internalRk)
  // Prefer `@friendlyName` over the raw template name so instantiations like
  // `CreateRequest<Plan>` become `createPlanRequest` instead of
  // `createRequest_2`. Mirrors the TS SDK emitter and the OpenAPI emitter,
  // which already honor the friendly name. References resolve via refkey, so
  // overriding the declaration name keeps call sites consistent.
  const friendlyName = getFriendlyName($.program, props.type)
  const newProps = ay.mergeProps(varDeclProps, {
    refkey: refkeys,
    name:
      props.name ||
      friendlyName ||
      ('name' in props.type &&
        typeof props.type.name === 'string' &&
        props.type.name) ||
      props.type.kind,
  })

  return (
    <DeclaringTypeContext.Provider value={props.type}>
      <ZodCustomTypeComponent
        declare
        type={props.type}
        Declaration={ts.VarDeclaration}
        declarationProps={newProps}
      >
        <ts.VarDeclaration {...newProps}>
          <ZodSchema {...zodSchemaProps} />
        </ts.VarDeclaration>
      </ZodCustomTypeComponent>
    </DeclaringTypeContext.Provider>
  )
}
