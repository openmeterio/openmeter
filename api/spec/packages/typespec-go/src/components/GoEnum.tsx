import * as ay from '@alloy-js/core'
import * as go from '@alloy-js/go'
import { type Refkey } from '@alloy-js/core'
import { type Union, type UnionVariant } from '@typespec/compiler'
import { goExportedName } from '../go-types.js'

export interface GoEnumProps {
  union: Union
  name: string
  refkey: Refkey
  doc?: string
}

export function GoEnum({ union, name, refkey, doc }: GoEnumProps) {
  const members = [...union.variants.values()].map((variant) => ({
    name: variantName(variant),
    value: variantValue(variant),
  }))

  return (
    <>
      <go.TypeDeclaration name={name} refkey={refkey} doc={doc}>
        string
      </go.TypeDeclaration>
      {'\n\n'}
      <go.VariableDeclarationGroup const>
        {members.map((member) => (
          <go.VariableDeclaration
            name={`${name}${goExportedName(member.name)}`}
            type={refkey}
          >
            {JSON.stringify(member.value)}
          </go.VariableDeclaration>
        ))}
      </go.VariableDeclarationGroup>
      {'\n\n'}
      <go.FunctionDeclaration
        name="Valid"
        receiver={<go.FunctionReceiver name="value" type={name} />}
        returns="bool"
      >
        {ay.code`
          switch value {
          case ${members
            .map((member) => `${name}${goExportedName(member.name)}`)
            .join(', ')}:
            return true
          default:
            return false
          }
        `}
      </go.FunctionDeclaration>
    </>
  )
}

function variantName(variant: UnionVariant): string {
  return typeof variant.name === 'symbol' ? variantValue(variant) : variant.name
}

function variantValue(variant: UnionVariant): string {
  if (variant.type.kind !== 'String') {
    throw new Error(
      `Go string enum variant ${String(variant.name)} is not a string literal`,
    )
  }

  return variant.type.value
}
