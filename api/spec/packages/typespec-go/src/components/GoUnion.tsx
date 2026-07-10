import * as ay from '@alloy-js/core'
import * as go from '@alloy-js/go'
import type {
  Model,
  ModelProperty,
  Program,
  Type,
  Union,
  UnionVariant,
} from '@typespec/compiler'
import { resolveEncodedName } from '@typespec/compiler'
import { $ } from '@typespec/compiler/typekit'
import { goExportedName, goType } from '../go-types.js'
import { json } from '../stdlib.js'

export interface GoUnionProps {
  program: Program
  union: Union
  name: string
  mode?: 'input' | 'output'
  doc?: string
}

/**
 * Emits a JSON-preserving tagged union with named, typed accessors for every
 * model variant. The raw payload is retained so unknown discriminator values
 * remain forward compatible and round-trip unchanged.
 */
export function GoUnion({ program, union, name, mode, doc }: GoUnionProps) {
  const modelVariants = [...union.variants.values()].flatMap((variant) =>
    variant.type.kind === 'Model' ? [variant.type] : [],
  )
  const discriminator = discriminatorProperty(program, union, modelVariants)
  if (!discriminator && modelVariants.length > 1) {
    throw new Error(
      `typespec-go: union ${name} has multiple object variants but no discriminator; add @discriminated so Go accessors can select variants safely`,
    )
  }
  const variants = [...union.variants.values()].flatMap((variant) => {
    if (variant.type.kind === 'Intrinsic') {
      return []
    }

    const mapped = goType(program, variant.type, { mode }).type
    return [
      {
        variant,
        type: variant.type,
        name:
          variant.type.kind === 'Model' && typeof mapped === 'string'
            ? mapped
            : variantAccessorName(program, name, variant),
        goType: mapped,
        discriminatorValue:
          variant.type.kind === 'Model'
            ? discriminatorLiteral(variant.type, discriminator)
            : undefined,
      },
    ]
  })
  const discriminatorField = discriminator
    ? goExportedName(discriminator.wireName.replace(/^\$/, ''))
    : 'Type'

  const contractDoc = [
    ...(doc ? [doc, ''] : []),
    `${name} is a JSON-preserving tagged union: its zero value marshals as JSON null, and values must be built with the ${name}From* constructors.`,
    ...(discriminator
      ? [
          `The exported ${discriminatorField} field is decode-side metadata; MarshalJSON round-trips the original payload and ignores writes to it.`,
        ]
      : []),
  ].join('\n')

  return (
    <ay.List joiner={'\n\n'}>
      <go.StructTypeDeclaration name={name} doc={contractDoc}>
        <ay.List hardline>
          {discriminator ? (
            <go.StructMember
              name={discriminatorField}
              type="string"
              tag={{ json: discriminator.wireName }}
            />
          ) : undefined}
          <go.StructMember name="raw" type={json.RawMessage} />
        </ay.List>
      </go.StructTypeDeclaration>
      <go.FunctionDeclaration
        name="UnmarshalJSON"
        receiver={
          <go.FunctionReceiver
            name="u"
            type={<go.Pointer>{name}</go.Pointer>}
          />
        }
        parameters={[{ name: 'data', type: '[]byte' }]}
        returns="error"
      >
        {discriminator
          ? ay.code`
              u.raw = append([]byte(nil), data...)
              if string(data) == "null" {
                u.${discriminatorField} = ""
                return nil
              }

              var envelope struct {
                Value string ${`\`json:${JSON.stringify(discriminator.wireName)}\``}
              }
              if err := ${json.Unmarshal}(data, &envelope); err != nil {
                return err
              }
              u.${discriminatorField} = envelope.Value
              return nil
            `
          : ay.code`
              u.raw = append([]byte(nil), data...)
              return nil
            `}
      </go.FunctionDeclaration>
      <go.FunctionDeclaration
        name="MarshalJSON"
        receiver={<go.FunctionReceiver name="u" type={name} />}
        returns={['[]byte', 'error']}
      >
        {ay.code`
          if len(u.raw) == 0 {
            return []byte("null"), nil
          }
          return append([]byte(nil), u.raw...), nil
        `}
      </go.FunctionDeclaration>
      <ay.List>
        {variants.flatMap((item) => [
          <go.FunctionDeclaration
            name={`As${item.name}`}
            receiver={<go.FunctionReceiver name="u" type={name} />}
            returns={[<go.Pointer>{item.goType}</go.Pointer>, 'error']}
          >
            {accessorBody({
              unionName: name,
              variantName: item.name,
              variantType: item.type,
              goType: item.goType,
              discriminator,
              discriminatorField,
              discriminatorValue: item.discriminatorValue,
            })}
          </go.FunctionDeclaration>,
          '\n\n',
          <go.FunctionDeclaration
            name={`${name}From${item.name}`}
            parameters={[{ name: 'value', type: item.goType }]}
            returns={[name, 'error']}
          >
            {discriminator && item.discriminatorValue !== undefined
              ? ay.code`
                  value.${discriminatorField} = ${JSON.stringify(item.discriminatorValue)}
                  raw, err := ${json.Marshal}(value)
                  if err != nil {
                    return ${name}{}, err
                  }
                  var result ${name}
                  if err := result.UnmarshalJSON(raw); err != nil {
                    return ${name}{}, err
                  }
                  return result, nil
                `
              : ay.code`
                  raw, err := ${json.Marshal}(value)
                  if err != nil {
                    return ${name}{}, err
                  }
                  var result ${name}
                  if err := result.UnmarshalJSON(raw); err != nil {
                    return ${name}{}, err
                  }
                  return result, nil
                `}
          </go.FunctionDeclaration>,
          '\n\n',
        ])}
      </ay.List>
    </ay.List>
  )
}

function accessorBody({
  unionName,
  variantName,
  variantType,
  goType: goTypeName,
  discriminator,
  discriminatorField,
  discriminatorValue,
}: {
  unionName: string
  variantName: string
  variantType: Type
  goType: ay.Children
  discriminator: { name: string; wireName: string } | undefined
  discriminatorField: string
  discriminatorValue: string | undefined
}): ay.Children {
  const discriminatorGuard =
    discriminator && discriminatorValue !== undefined
      ? ay.code`
          if u.${discriminatorField} != ${JSON.stringify(discriminatorValue)} {
            return nil, ${go.std.fmt.Errorf}("${unionName}: expected ${discriminator.wireName} %q, got %q", ${JSON.stringify(discriminatorValue)}, u.${discriminatorField})
          }
        `
      : undefined
  const scalarGuard = variantValidation(unionName, variantName, variantType)

  if (discriminatorGuard && scalarGuard) {
    return ay.code`
      ${discriminatorGuard}
      var value ${goTypeName}
      if err := ${json.Unmarshal}(u.raw, &value); err != nil {
        return nil, err
      }
      ${scalarGuard}
      return &value, nil
    `
  }

  if (discriminatorGuard) {
    return ay.code`
      ${discriminatorGuard}
      var value ${goTypeName}
      if err := ${json.Unmarshal}(u.raw, &value); err != nil {
        return nil, err
      }
      return &value, nil
    `
  }

  if (scalarGuard) {
    return ay.code`
      var value ${goTypeName}
      if err := ${json.Unmarshal}(u.raw, &value); err != nil {
        return nil, err
      }
      ${scalarGuard}
      return &value, nil
    `
  }

  return ay.code`
    var value ${goTypeName}
    if err := ${json.Unmarshal}(u.raw, &value); err != nil {
      return nil, err
    }
    return &value, nil
  `
}

export function variantAccessorName(
  program: Program,
  unionName: string,
  variant: UnionVariant,
): string {
  const variantName =
    typeof variant.name === 'symbol'
      ? `Variant${goExportedName(String(variant.name.description ?? ''))}`
      : goExportedName(variant.name)
  return variantName || `${unionName}Variant`
}

export function discriminatorProperty(
  program: Program,
  union: Union,
  variants: Model[],
): { name: string; wireName: string } | undefined {
  if (variants.length === 0) {
    return undefined
  }

  const discriminated = $(program).union.getDiscriminatedUnion(union)
  if (discriminated) {
    if (discriminated.options.envelope !== 'none') {
      throw new Error(
        `typespec-go: union ${union.name ?? '<anonymous union>'} uses unsupported discriminated union envelope ${discriminated.options.envelope}`,
      )
    }
    const name = discriminated.options.discriminatorPropertyName
    const property = variants[0]?.properties.get(name)
    return {
      name,
      wireName: property
        ? resolveEncodedName(
            program,
            property as ModelProperty & { name: string },
            'application/json',
          )
        : name,
    }
  }

  for (const candidate of ['type', '$type']) {
    if (
      variants.length > 0 &&
      variants.every((model) => model.properties.has(candidate))
    ) {
      const property = variants[0]!.properties.get(candidate)!
      return {
        name: candidate,
        wireName: resolveEncodedName(program, property, 'application/json'),
      }
    }
  }

  return undefined
}

export function discriminatorLiteral(
  model: Model,
  discriminator: { name: string; wireName: string } | undefined,
): string | undefined {
  if (!discriminator) {
    return undefined
  }

  const property = model.properties.get(discriminator.name)
  if (!property) {
    return undefined
  }

  switch (property.type.kind) {
    case 'String':
      return property.type.value
    case 'EnumMember':
      return String(property.type.value ?? property.type.name)
    default:
      return undefined
  }
}

function variantValidation(
  unionName: string,
  variantName: string,
  type: Type,
): ay.Children {
  switch (type.kind) {
    case 'Enum':
      return ay.code`
        if !value.Valid() {
          return nil, ${go.std.fmt.Errorf}("${unionName}: value %q is not ${variantName}", value)
        }
      `
    case 'EnumMember':
      return ay.code`
        if value != ${JSON.stringify(type.value ?? type.name)} {
          return nil, ${go.std.fmt.Errorf}("${unionName}: value %q is not ${variantName}", value)
        }
      `
    case 'String':
    case 'Number':
    case 'Boolean':
      if ('value' in type && type.value !== undefined) {
        return ay.code`
          if value != ${JSON.stringify(type.value)} {
            return nil, ${go.std.fmt.Errorf}("${unionName}: value %q is not ${variantName}", value)
          }
        `
      }
      return undefined
    default:
      return undefined
  }
}
