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
  // A union-typed variant (`Invoice | InvoiceReference`, where `Invoice` is the
  // discriminated union of its concrete types) contributes its own models to the
  // same ambiguity: the outer union has no discriminator, so an As* accessor
  // decodes the preserved payload whichever side produced it.
  const nestedVariants = nestedUnionModelVariants(union)
  const discriminator = discriminatorProperty(program, union, modelVariants)
  if (!discriminator && modelVariants.length + nestedVariants.length > 1) {
    // Without a discriminator an As* accessor cannot verify it was handed its
    // own variant — it just unmarshals the preserved payload into the variant
    // type. That is still faithful when the variants agree on every JSON key
    // they share and differ only in which keys they declare (a reference model
    // versus the full resource): each accessor then reads exactly the subset of
    // the payload its type describes. A key with two different types across
    // variants would decode into the wrong Go type, so that still has to be
    // discriminated.
    const conflict = conflictingVariantProperty(
      program,
      modelVariants,
      nestedVariants,
    )
    if (conflict) {
      throw new Error(
        `typespec-go: union ${name} has object variants that disagree on ${conflict}; add @discriminated so Go accessors can select variants safely (variants whose properties are a subset of another's are supported)`,
      )
    }
  }
  // The discriminated path assumes every selectable variant is a model: the
  // From* constructors envelope the payload under the discriminator property
  // and the As* accessors match on it. A scalar or enum variant has neither,
  // so it would emit constructors/accessors that cannot round-trip.
  if (discriminator) {
    const nonModelVariants = [...union.variants.values()].filter(
      (variant) =>
        variant.type.kind !== 'Model' && variant.type.kind !== 'Intrinsic',
    )
    if (nonModelVariants.length > 0) {
      throw new Error(
        `typespec-go: discriminated union ${name} mixes model and non-model variants (${nonModelVariants
          .map((variant) => variant.type.kind)
          .join(', ')}); only model variants can carry the discriminator`,
      )
    }
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
        name: variantGoName(program, name, variant, mode),
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

/**
 * The first JSON key two model variants declare with different types, described
 * for the build error — or undefined when the variants only differ in which keys
 * they declare, which every accessor can decode from the same payload.
 *
 * `nestedVariants` are the models a union-typed variant contributes. They are
 * checked against `variants` but never against each other, and never added to
 * the comparison map: a nested DISCRIMINATED union's models legitimately disagree
 * on non-discriminator keys, because that union selects on its discriminator
 * rather than by shape. Comparing them with each other would fail the build on
 * unions that decode correctly.
 */
export function conflictingVariantProperty(
  program: Program,
  variants: Model[],
  nestedVariants: Model[] = [],
): string | undefined {
  const byKey = new Map<string, { type: Type; variant: string }>()
  for (const model of variants) {
    for (const [key, property] of variantProperties(program, model)) {
      const seen = byKey.get(key)
      if (!seen) {
        byKey.set(key, { type: property.type, variant: model.name })
        continue
      }
      if (seen.type !== property.type) {
        return `"${key}" (different types in ${seen.variant} and ${model.name})`
      }
    }
  }

  for (const model of nestedVariants) {
    for (const [key, property] of variantProperties(program, model)) {
      const seen = byKey.get(key)
      if (seen && seen.type !== property.type) {
        return `"${key}" (different types in ${seen.variant} and ${model.name})`
      }
    }
  }
  return undefined
}

/**
 * The model variants reachable through a union's union-typed variants, at any
 * nesting depth. Cycle-guarded, so a self-referential union contributes each of
 * its models once instead of recursing forever.
 */
export function nestedUnionModelVariants(
  union: Union,
  seen = new Set<Union>(),
): Model[] {
  const models: Model[] = []
  for (const variant of union.variants.values()) {
    const nested = variant.type
    if (nested.kind !== 'Union' || seen.has(nested)) {
      continue
    }
    seen.add(nested)
    for (const nestedVariant of nested.variants.values()) {
      if (nestedVariant.type.kind === 'Model') {
        models.push(nestedVariant.type)
      }
    }
    models.push(...nestedUnionModelVariants(nested, seen))
  }
  return models
}

/**
 * A model's properties keyed by their JSON name, including inherited ones, with
 * an override shadowing the property it replaces.
 */
function variantProperties(
  program: Program,
  model: Model,
): Map<string, ModelProperty> {
  const properties = new Map<string, ModelProperty>()
  for (
    let current: Model | undefined = model;
    current;
    current = current.baseModel
  ) {
    for (const property of current.properties.values()) {
      const key = resolveEncodedName(
        program,
        property as ModelProperty & { name: string },
        'application/json',
      )
      if (!properties.has(key)) {
        properties.set(key, property)
      }
    }
  }
  return properties
}

/**
 * The Go name a union member's accessors carry (AsX / FromX / XFromY). A variant
 * whose type is a named model or a named union uses that type's mapped Go name —
 * a nested union variant (`Invoice` inside `InvoiceOrReference`) must not fall
 * through to the anonymous-variant fallback, which would emit a placeholder like
 * `AsVariantName` into the published SDK surface.
 */
export function variantGoName(
  program: Program,
  unionName: string,
  variant: UnionVariant,
  mode?: 'input' | 'output',
): string {
  const mapped = goType(program, variant.type, { mode }).type
  if (
    (variant.type.kind === 'Model' || variant.type.kind === 'Union') &&
    typeof mapped === 'string'
  ) {
    return mapped
  }
  return variantAccessorName(program, unionName, variant)
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
