import * as ay from '@alloy-js/core'
import * as go from '@alloy-js/go'
import type { Enum } from '@typespec/compiler'
import { goExportedName } from '../go-types.js'

export function GoTypeSpecEnum({
  enumType,
  name,
  doc,
}: {
  enumType: Enum
  name: string
  doc?: string
}) {
  const members = [...enumType.members.values()].map((member) => ({
    name: `${name}${goExportedName(member.name)}`,
    value: member.value ?? member.name,
  }))
  const underlying = members.some((member) => typeof member.value === 'number')
    ? 'int'
    : 'string'

  return (
    <>
      <go.TypeDeclaration name={name} doc={doc}>
        {underlying}
      </go.TypeDeclaration>
      {'\n\n'}
      <go.VariableDeclarationGroup const>
        <ay.List hardline>
          {members.map((member) => (
            <go.VariableDeclaration name={member.name} type={name}>
              {JSON.stringify(member.value)}
            </go.VariableDeclaration>
          ))}
        </ay.List>
      </go.VariableDeclarationGroup>
      {'\n\n'}
      <go.FunctionDeclaration
        name="Valid"
        receiver={<go.FunctionReceiver name="value" type={name} />}
        returns="bool"
      >
        {ay.code`
          switch value {
          case ${members.map((member) => member.name).join(', ')}:
            return true
          default:
            return false
          }
        `}
      </go.FunctionDeclaration>
    </>
  )
}
