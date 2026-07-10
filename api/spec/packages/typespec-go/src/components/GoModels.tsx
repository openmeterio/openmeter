import * as ay from '@alloy-js/core'
import * as go from '@alloy-js/go'
import { type Program, type Type, type Union } from '@typespec/compiler'
import { $ } from '@typespec/compiler/typekit'
import {
  goFields,
  goProjectionsOf,
  goType,
  nullableUnionElement,
  optionalTypeName,
  typeName,
  type GoDeclarationPlan,
} from '../go-types.js'
import { GoEnum } from './GoEnum.js'
import { GoStruct } from './GoStruct.js'
import { GoTypeSpecEnum } from './GoTypeSpecEnum.js'
import { GoUnion } from './GoUnion.js'
import { isRuntimeBackedTypeName } from '../runtime-symbols.js'

export interface GoModelsProps {
  program: Program
  types: Set<Type>
}

export function GoModels({ program, types }: GoModelsProps) {
  const typekit = $(program)
  const projections = goProjectionsOf(program)
  const emittedNames = new Map<string, Type>()
  const declarations = [...types]
    .filter(
      (type) =>
        type.kind !== 'Model' ||
        (!typekit.array.is(type) && !typekit.record.is(type)),
    )
    .flatMap((type) => {
      const name = optionalTypeName(program, type)
      return name ? [{ name, type }] : []
    })
    .filter(({ name, type }) => !isRuntimeBackedTypeName(name, type.kind))
    .filter(({ name, type }) => {
      const existing = emittedNames.get(name)
      if (!existing) {
        emittedNames.set(name, type)
      }
      return !existing
    })
    // Code-point comparison, not localeCompare: the output is a committed
    // artifact, and locale-dependent ordering would produce spurious diffs
    // across machines.
    .sort((left, right) =>
      left.name < right.name ? -1 : left.name > right.name ? 1 : 0,
    )

  // Without a configured projection registry (unit tests), every model emits
  // its read projection under the natural name.
  const plannedDeclarations = (
    type: Type,
    name: string,
  ): GoDeclarationPlan[] =>
    projections
      ? (projections.declarations.get(type) ?? [])
      : [{ name, mode: 'read' }]

  return (
    <ay.List joiner={'\n\n'}>
      {declarations.map(({ name, type }) => {
        switch (type.kind) {
          case 'Model': {
            const doc = typekit.type.getDoc(type)
            return (
              <ay.List joiner={'\n\n'}>
                {plannedDeclarations(type, name).map((declaration) => (
                  <GoStruct
                    name={declaration.name}
                    fields={goFields(program, type, {
                      mode: declaration.mode === 'input' ? 'input' : undefined,
                    })}
                    doc={doc}
                  />
                ))}
              </ay.List>
            )
          }
          case 'Union':
            return (
              <ay.List joiner={'\n\n'}>
                {plannedDeclarations(type, name).map((declaration) =>
                  renderUnion(program, type, declaration),
                )}
              </ay.List>
            )
          case 'Enum':
            return (
              <GoTypeSpecEnum
                enumType={type}
                name={name}
                doc={typekit.type.getDoc(type)}
              />
            )
          // Scalars never carry generated behavior and no emitted field
          // references a scalar alias (fields use the underlying Go type),
          // so declaring them would only add dead exported names.
          case 'Scalar':
          default:
            return undefined
        }
      })}
    </ay.List>
  )
}

export function renderUnion(
  program: Program,
  union: Union,
  declaration: GoDeclarationPlan,
) {
  const name = declaration.name
  const mode = declaration.mode === 'input' ? ('input' as const) : undefined
  const doc = $(program).type.getDoc(union)

  if (nullableUnionElement(union)) {
    return (
      <go.TypeDeclaration name={name} alias doc={doc}>
        {goType(program, union, { mode }).type}
      </go.TypeDeclaration>
    )
  }

  const variants = [...union.variants.values()]
  if (
    variants.length > 0 &&
    variants.every((variant) => variant.type.kind === 'String')
  ) {
    return <GoEnum union={union} name={name} refkey={ay.refkey()} doc={doc} />
  }

  if (
    variants.length > 0 &&
    variants.every((variant) => isStringLike(program, variant.type))
  ) {
    return (
      <go.TypeDeclaration name={name} doc={doc}>
        string
      </go.TypeDeclaration>
    )
  }

  if (
    variants.length > 0 &&
    variants.every((variant) => variant.type.kind === 'Model')
  ) {
    return (
      <GoUnion
        program={program}
        union={union}
        name={name}
        mode={mode}
        doc={doc}
      />
    )
  }

  const concrete = variants.filter(
    (variant) => variant.type.kind !== 'Intrinsic',
  )
  if (concrete.length === 1) {
    return (
      <go.TypeDeclaration name={name} alias doc={doc}>
        {goType(program, concrete[0]!.type, { mode }).type}
      </go.TypeDeclaration>
    )
  }

  if (concrete.length > 1) {
    return (
      <GoUnion
        program={program}
        union={union}
        name={name}
        mode={mode}
        doc={doc}
      />
    )
  }

  throw new Error(
    `typespec-go: union ${typeName(program, union)} has no concrete variants representable in Go; give it at least one non-intrinsic variant or replace the union with the intended model before emitting it`,
  )
}

export function isStringLike(program: Program, type: Type): boolean {
  switch (type.kind) {
    case 'String':
      return true
    case 'Scalar':
      return goType(program, type).type === 'string'
    case 'Enum':
      return [...type.members.values()].every(
        (member) => typeof (member.value ?? member.name) === 'string',
      )
    case 'EnumMember':
      return typeof (type.value ?? type.name) === 'string'
    default:
      return false
  }
}
