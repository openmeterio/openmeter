import * as ay from '@alloy-js/core'
import * as go from '@alloy-js/go'
import type { Refkey } from '@alloy-js/core'
import type { GoField } from '../go-types.js'

export interface GoStructProps {
  name: string
  fields: GoField[]
  refkey?: Refkey
  doc?: string
  tags?: boolean
}

export function GoStruct({
  name,
  fields,
  refkey,
  doc,
  tags = true,
}: GoStructProps) {
  return (
    <go.StructTypeDeclaration name={name} refkey={refkey} doc={doc}>
      <ay.List hardline>
        {fields.map((field) => (
          <go.StructMember
            name={field.name}
            type={field.type}
            doc={field.doc}
            tag={
              tags
                ? {
                    json: `${field.wireName}${field.optional ? ',omitempty' : ''}`,
                  }
                : undefined
            }
          />
        ))}
      </ay.List>
    </go.StructTypeDeclaration>
  )
}
