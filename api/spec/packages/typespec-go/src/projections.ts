import {
  resolveEncodedName,
  walkPropertiesInherited,
  type Model,
  type Operation,
  type Program,
  type Type,
  type Union,
} from '@typespec/compiler'
import { $ } from '@typespec/compiler/typekit'
import { isHeader, isStatusCode, isVisible, Visibility } from '@typespec/http'
import {
  goExportedName,
  goType,
  inputVariantName,
  nullableUnionElement,
  oneOrManyElement,
  optionalTypeName,
  runtimeFilterTypeName,
  type GoDeclarationPlan,
  type GoPayloadMode,
} from './go-types.js'
import { isStringLike } from './components/GoModels.js'
import {
  discriminatorLiteral,
  discriminatorProperty,
  variantAccessorName,
} from './components/GoUnion.js'
import { isRuntimeBackedTypeName } from './runtime-symbols.js'
import { describeOperations } from './operations.js'

export interface GoReachability {
  /** Types reachable from response bodies, path params, and query params. */
  readReachable: Set<Type>
  /** Types reachable from request bodies. */
  inputReachable: Set<Type>
  /** Every type each resource touches, for output file grouping. */
  byResource: Map<string, Set<Type>>
}

/**
 * Walks every operation's payloads into read-side and input-side type sets.
 *
 * The read walk skips properties not visible in Lifecycle.Read: the server
 * never returns them, so their types must not be dragged into read models.
 * The input walk keeps every property because request trees are already
 * visibility-projected by the spec (Create/Update request templates).
 */
export function computeReachability(
  program: Program,
  groups: Iterable<readonly [string, Operation[]]>,
  bodyOverrides: Map<string, Type>,
): GoReachability {
  const readReachable = new Set<Type>()
  const inputReachable = new Set<Type>()
  const byResource = new Map<string, Set<Type>>()

  const makeWalker = (
    resourceTypes: Set<Type>,
    target: Set<Type>,
    filterRead: boolean,
  ) => {
    // Each direction keeps its own visited set: a type reachable from both a
    // request body and a response must land in both classification sets, not
    // just whichever walk happened to run first.
    const visited = new Set<Type>()
    const visit = (type: Type | undefined): void => {
      if (!type || visited.has(type)) {
        return
      }
      visited.add(type)
      resourceTypes.add(type)
      target.add(type)

      switch (type.kind) {
        case 'Model':
          if (type.baseModel) {
            visit(type.baseModel)
          }
          if (type.indexer) {
            visit(type.indexer.value)
          }
          for (const property of type.properties.values()) {
            if (filterRead && !isVisible(program, property, Visibility.Read)) {
              continue
            }
            visit(property.type)
          }
          break
        case 'Union':
          // Runtime-backed filter unions render as static runtime types
          // (StringFilter, DateTimeFilter, ...); walking their variants would
          // promote their anonymous object variants into dead declarations.
          if (type.name && isRuntimeBackedTypeName(type.name, type.kind)) {
            break
          }
          for (const variant of type.variants.values()) {
            visit(variant.type)
          }
          break
        case 'Tuple':
          for (const value of type.values) {
            visit(value)
          }
          break
        case 'EnumMember':
          visit(type.enum)
          break
        default:
          break
      }
    }
    return visit
  }

  for (const [resource, operations] of groups) {
    const resourceTypes = new Set<Type>()
    byResource.set(resource, resourceTypes)
    const visitRead = makeWalker(resourceTypes, readReachable, true)
    const visitInput = makeWalker(resourceTypes, inputReachable, false)

    for (const operation of describeOperations(
      program,
      resource,
      operations,
      bodyOverrides,
    )) {
      for (const parameter of operation.pathParams) {
        visitRead(parameter.type)
      }
      for (const parameter of operation.queryParams) {
        if (parameter.name === 'filter' && parameter.type.kind === 'Model') {
          // Deep-object filter models are rendered as query params structs by
          // GoResource, not as JSON models; only their field types are models.
          for (const property of parameter.type.properties.values()) {
            visitRead(property.type)
          }
        } else {
          visitRead(parameter.type)
        }
      }
      visitRead(operation.response)
      visitInput(operation.body)
    }
  }

  return { readReachable, inputReachable, byResource }
}

/**
 * Types whose input projection renders differently from their read projection,
 * limited to types reachable from both directions (only those emit twice).
 *
 * Divergence causes: a defaulted property (optional in input), an optional
 * collection (pointered in input so an explicitly empty collection survives
 * omitempty), a property not visible in Lifecycle.Read (dropped from the read
 * projection), or a reference to another divergent both-reachable type.
 */
export function computeDivergentTypes(
  program: Program,
  readReachable: Set<Type>,
  inputReachable: Set<Type>,
): Set<Type> {
  const typekit = $(program)
  const divergent = new Set<Type>()
  const memo = new Map<Type, boolean>()
  const visiting = new Set<Type>()
  const both = (type: Type): boolean =>
    readReachable.has(type) && inputReachable.has(type)

  const isCollection = (type: Type): boolean =>
    type.kind === 'Model' && (typekit.array.is(type) || typekit.record.is(type))

  // Whether a field or variant referencing this type renders a different Go
  // type name in input context than in read context.
  const referenceDiverges = (type: Type): boolean => {
    switch (type.kind) {
      case 'Model':
        if (typekit.array.is(type) || typekit.record.is(type)) {
          return type.indexer ? referenceDiverges(type.indexer.value) : false
        }
        return both(type) && diverges(type)
      case 'Union':
        if (optionalTypeName(program, type)) {
          return both(type) && diverges(type)
        }
        return [...type.variants.values()].some((variant) =>
          referenceDiverges(variant.type),
        )
      case 'Tuple':
        return type.values.some(referenceDiverges)
      default:
        return false
    }
  }

  const diverges = (type: Model | Union): boolean => {
    const cached = memo.get(type)
    if (cached !== undefined) {
      return cached
    }
    if (visiting.has(type)) {
      return false
    }
    visiting.add(type)

    let result = false
    if (type.kind === 'Model') {
      for (const property of walkPropertiesInherited(type)) {
        if (isHeader(program, property) || isStatusCode(program, property)) {
          continue
        }
        if (
          !isVisible(program, property, Visibility.Read) ||
          // A default only changes the input rendering when the property is
          // required: fieldShape keeps an already-optional property optional
          // in both modes, so the projections stay byte-identical.
          (property.defaultValue !== undefined && !property.optional) ||
          (property.optional && isCollection(property.type)) ||
          referenceDiverges(property.type)
        ) {
          result = true
          break
        }
      }
      if (!result && type.indexer) {
        result = referenceDiverges(type.indexer.value)
      }
    } else {
      result = [...type.variants.values()].some((variant) =>
        referenceDiverges(variant.type),
      )
    }

    visiting.delete(type)
    memo.set(type, result)
    if (result) {
      divergent.add(type)
    }
    return result
  }

  for (const type of readReachable) {
    if (
      (type.kind === 'Model' || type.kind === 'Union') &&
      inputReachable.has(type)
    ) {
      diverges(type)
    }
  }

  // Only both-reachable types emit two projections; drop incidental members.
  for (const type of [...divergent]) {
    if (!both(type)) {
      divergent.delete(type)
    }
  }

  return divergent
}

/**
 * Assigns deterministic Go type names to anonymous models reachable from the
 * public surface, derived from the enclosing type plus the field (for example
 * SubscriptionCreate.customer becomes SubscriptionCreateCustomer). Without
 * promotion a required anonymous struct field cannot be populated in a
 * composite literal without redeclaring the whole anonymous type.
 */
export function promoteAnonymousModels(
  program: Program,
  types: Set<Type>,
): Map<Type, string> {
  const typekit = $(program)
  const promoted = new Map<Type, string>()
  const taken = new Map<string, Type>()
  for (const type of types) {
    const name = optionalTypeName(program, type)
    if (name) {
      taken.set(name, type)
    }
  }

  const isAnonymousStruct = (type: Type): type is Model =>
    type.kind === 'Model' &&
    !typekit.array.is(type) &&
    !typekit.record.is(type) &&
    optionalTypeName(program, type) === undefined &&
    promoted.get(type) === undefined

  const claim = (type: Model, name: string): void => {
    const existing = taken.get(name)
    if (existing && existing !== type) {
      throw new Error(
        `typespec-go: promoted anonymous model name ${name} collides with an existing ${existing.kind}; add a @friendlyName to disambiguate`,
      )
    }
    promoted.set(type, name)
    taken.set(name, type)
    visitModel(type, name)
  }

  const visitFieldType = (type: Type, parentName: string, field: string) => {
    switch (type.kind) {
      case 'Model':
        if (typekit.array.is(type) || typekit.record.is(type)) {
          if (type.indexer) {
            visitFieldType(type.indexer.value, parentName, field)
          }
          return
        }
        if (isAnonymousStruct(type)) {
          claim(type, `${parentName}${goExportedName(field)}`)
        }
        return
      case 'Union':
        for (const variant of type.variants.values()) {
          visitFieldType(variant.type, parentName, field)
        }
        return
      case 'Tuple':
        for (const value of type.values) {
          visitFieldType(value, parentName, field)
        }
        return
      default:
        return
    }
  }

  const visitModel = (model: Model, name: string): void => {
    for (const property of walkPropertiesInherited(model)) {
      if (isHeader(program, property) || isStatusCode(program, property)) {
        continue
      }
      visitFieldType(property.type, name, property.name)
    }
    if (model.indexer) {
      visitFieldType(model.indexer.value, name, 'Item')
    }
  }

  const named = [...types]
    .flatMap((type) => {
      if (type.kind !== 'Model' && type.kind !== 'Union') {
        return []
      }
      if (
        type.kind === 'Model' &&
        (typekit.array.is(type) || typekit.record.is(type))
      ) {
        return []
      }
      const name = optionalTypeName(program, type)
      return name ? [{ name, type }] : []
    })
    .sort((left, right) =>
      left.name < right.name ? -1 : left.name > right.name ? 1 : 0,
    )

  for (const { name, type } of named) {
    if (type.kind === 'Model') {
      visitModel(type, name)
    } else {
      for (const variant of type.variants.values()) {
        const field =
          typeof variant.name === 'symbol'
            ? String(variant.name.description ?? 'Variant')
            : variant.name
        visitFieldType(variant.type, name, field)
      }
    }
  }

  return promoted
}

/**
 * Which projections each reachable type emits, and under what names.
 *
 * A type reachable only from requests emits only its input projection under
 * the natural name; only from responses, only the read projection. A type
 * reachable from both emits one declaration when the projections agree, and a
 * read declaration plus a NameInput declaration when they diverge.
 */
export function planDeclarations(
  program: Program,
  types: Set<Type>,
  readReachable: Set<Type>,
  inputReachable: Set<Type>,
  divergent: Set<Type>,
): Map<Type, GoDeclarationPlan[]> {
  const typekit = $(program)
  const plan = new Map<Type, GoDeclarationPlan[]>()

  for (const type of types) {
    if (type.kind !== 'Model' && type.kind !== 'Union') {
      continue
    }
    if (
      type.kind === 'Model' &&
      (typekit.array.is(type) || typekit.record.is(type))
    ) {
      continue
    }
    const name = optionalTypeName(program, type)
    if (!name || isRuntimeBackedTypeName(name, type.kind)) {
      continue
    }

    const needsRead = readReachable.has(type)
    const needsInput = inputReachable.has(type)
    if (needsInput && !needsRead) {
      plan.set(type, [{ name, mode: 'input' }])
    } else if (needsRead && needsInput && divergent.has(type)) {
      plan.set(type, [
        { name, mode: 'read' },
        { name: inputVariantName(name), mode: 'input' },
      ])
    } else {
      plan.set(type, [{ name, mode: 'read' }])
    }
  }

  return plan
}

const PROJECTION_PREFIXES = ['Create', 'Update', 'Upsert'] as const

interface PlannedDeclaration {
  type: Type
  declaration: GoDeclarationPlan
}

/**
 * Collapses visibility-projection twins onto their canonical types.
 *
 * The spec's Create/Update request templates copy every nested model (and
 * union) into a prefixed twin even when visibility filtering removes nothing,
 * leaving byte-identical duplicates such as UpdateAddress next to Address.
 * A declaration whose name is Create/Update/Upsert + the name of another
 * emitted declaration (matched by emitted Go name, or by the source's declared
 * TypeSpec name when the canonical emits under a @friendlyName), and whose
 * rendered shape is identical, is dropped with every reference redirected to
 * the canonical name — so read-modify-write flows need no type mapping.
 *
 * Matching is a recursive structural comparison rather than flat text
 * equality: when two field references differ only because the candidate side
 * points at another prefixed twin of the target side's type (UpdateRateCard-
 * TaxConfig.code is UpdateResourceReference where RateCardTaxConfig.code is
 * TaxCodeReference), that reference pair is matched recursively and committed
 * as an additional alias when the enclosing declarations match. The outer
 * loop runs to a fixpoint because twins reference other twins.
 *
 * Returns aliases keyed by the dropped declaration name; the corresponding
 * plan entries are removed in place.
 */
export function computeStructuralAliases(
  program: Program,
  plan: Map<Type, GoDeclarationPlan[]>,
  readReachable: Set<Type>,
  divergent: Set<Type>,
): Map<string, string> {
  const typekit = $(program)
  const aliases = new Map<string, string>()

  const declarationsByName = new Map<string, PlannedDeclaration>()
  for (const [type, declarations] of plan) {
    for (const declaration of declarations) {
      declarationsByName.set(declaration.name, { type, declaration })
    }
  }

  // FilterVisibility twins are named from the source's declared TypeSpec name,
  // while the canonical type may emit under a @friendlyName (CreateCurrencyCode
  // vs BillingCurrencyCode); index declared names so those still resolve. An
  // ambiguous declared name (every ResourceReference<T> instantiation declares
  // "ResourceReference") yields no top-level target, but such twins still
  // collapse as reference pairs inside an enclosing declaration match, where
  // the target instantiation is known from context.
  const byDeclaredName = new Map<string, PlannedDeclaration | 'ambiguous'>()
  for (const [type, declarations] of plan) {
    const declared =
      'name' in type && typeof type.name === 'string' ? type.name : undefined
    if (!declared) {
      continue
    }
    const natural = declarations.find(
      (declaration) => declaration.name === optionalTypeName(program, type),
    )
    if (!natural) {
      continue
    }
    byDeclaredName.set(
      declared,
      byDeclaredName.has(declared)
        ? 'ambiguous'
        : { type, declaration: natural },
    )
  }

  const resolveThrough = (
    name: string,
    pending: Map<string, string>,
  ): string => {
    let final = name
    for (
      let next = aliases.get(final) ?? pending.get(final);
      next !== undefined;
      next = aliases.get(final) ?? pending.get(final)
    ) {
      final = next
    }
    return final
  }

  const referenceName = (
    type: Type,
    mode: GoPayloadMode,
    pending: Map<string, string>,
  ): string | undefined => {
    const name = optionalTypeName(program, type)
    if (!name) {
      return undefined
    }
    const runtime = type.kind === 'Union' ? runtimeFilterTypeName(name) : name
    if (runtime !== name) {
      return runtime
    }
    const base =
      mode === 'input' && divergent.has(type) ? inputVariantName(name) : name
    return resolveThrough(base, pending)
  }

  const projectionPrefixRest = (name: string): string | undefined => {
    for (const prefix of PROJECTION_PREFIXES) {
      if (
        name.startsWith(prefix) &&
        name.length > prefix.length &&
        /[A-Z]/.test(name[prefix.length]!)
      ) {
        return name.slice(prefix.length)
      }
    }
    return undefined
  }

  const scalarText = (type: Type): string | undefined => {
    try {
      return goType(program, type).text
    } catch {
      return undefined
    }
  }

  // Whether referencing `left` in `leftMode` renders the same Go type
  // expression as referencing `right` in `rightMode`, growing `pending` with
  // the reference-pair aliases the match depends on.
  const typesMatch = (
    left: Type,
    leftMode: GoPayloadMode,
    right: Type,
    rightMode: GoPayloadMode,
    pending: Map<string, string>,
    inProgress: Set<string>,
  ): boolean => {
    if (left.kind === 'Model' || right.kind === 'Model') {
      if (left.kind !== 'Model' || right.kind !== 'Model') {
        return false
      }
      const leftArray = typekit.array.is(left)
      const rightArray = typekit.array.is(right)
      const leftRecord = typekit.record.is(left)
      const rightRecord = typekit.record.is(right)
      if (leftArray !== rightArray || leftRecord !== rightRecord) {
        return false
      }
      if (leftArray || leftRecord) {
        return (
          left.indexer !== undefined &&
          right.indexer !== undefined &&
          typesMatch(
            left.indexer.value,
            leftMode,
            right.indexer.value,
            rightMode,
            pending,
            inProgress,
          )
        )
      }
      return namedReferencesMatch(
        left,
        leftMode,
        right,
        rightMode,
        pending,
        inProgress,
      )
    }

    if (left.kind === 'Union' || right.kind === 'Union') {
      if (left.kind !== 'Union' || right.kind !== 'Union') {
        return false
      }
      const leftNullable = nullableUnionElement(left)
      const rightNullable = nullableUnionElement(right)
      if ((leftNullable === undefined) !== (rightNullable === undefined)) {
        return false
      }
      if (leftNullable && rightNullable) {
        return typesMatch(
          leftNullable,
          leftMode,
          rightNullable,
          rightMode,
          pending,
          inProgress,
        )
      }
      const leftOneOrMany = oneOrManyElement(program, left)
      const rightOneOrMany = oneOrManyElement(program, right)
      if ((leftOneOrMany === undefined) !== (rightOneOrMany === undefined)) {
        return false
      }
      if (leftOneOrMany && rightOneOrMany) {
        return typesMatch(
          leftOneOrMany,
          leftMode,
          rightOneOrMany,
          rightMode,
          pending,
          inProgress,
        )
      }
      const leftNamed = optionalTypeName(program, left) !== undefined
      const rightNamed = optionalTypeName(program, right) !== undefined
      if (leftNamed && rightNamed) {
        return namedReferencesMatch(
          left,
          leftMode,
          right,
          rightMode,
          pending,
          inProgress,
        )
      }
      if (leftNamed || rightNamed) {
        return false
      }
      // Anonymous unions render through goType's fallbacks; both sides must
      // reduce to the same plain text (single concrete variant or string set).
      const leftText = scalarText(left)
      return leftText !== undefined && leftText === scalarText(right)
    }

    const leftText = scalarText(left)
    return leftText !== undefined && leftText === scalarText(right)
  }

  const namedReferencesMatch = (
    left: Type,
    leftMode: GoPayloadMode,
    right: Type,
    rightMode: GoPayloadMode,
    pending: Map<string, string>,
    inProgress: Set<string>,
  ): boolean => {
    const leftName = referenceName(left, leftMode, pending)
    const rightName = referenceName(right, rightMode, pending)
    if (leftName === undefined || rightName === undefined) {
      return false
    }
    if (leftName === rightName) {
      return true
    }

    // Prefix tolerance: the candidate side may reference its own prefixed
    // twin of the type the target side references.
    const rest = projectionPrefixRest(leftName)
    if (rest === undefined) {
      return false
    }
    const rightDeclared =
      'name' in right && typeof right.name === 'string' ? right.name : undefined
    if (rest !== rightName && rest !== rightDeclared) {
      return false
    }
    const leftDeclaration = plan
      .get(left)
      ?.find((declaration) => declaration.name === leftName)
    const rightDeclaration = plan
      .get(right)
      ?.find((declaration) => declaration.name === rightName)
    if (!leftDeclaration || !rightDeclaration) {
      return false
    }
    const key = `${leftName}->${rightName}`
    if (inProgress.has(key)) {
      return true
    }
    inProgress.add(key)
    const matched = declarationsMatch(
      left,
      leftDeclaration.mode,
      right,
      rightDeclaration.mode,
      pending,
      inProgress,
    )
    inProgress.delete(key)
    if (matched) {
      pending.set(leftName, rightName)
    }
    return matched
  }

  const filteredProperties = (type: Model, mode: GoPayloadMode) =>
    [...walkPropertiesInherited(type)].filter(
      (property) =>
        !isHeader(program, property) &&
        !isStatusCode(program, property) &&
        (mode === 'input' ||
          !readReachable.has(type) ||
          isVisible(program, property, Visibility.Read)),
    )

  const fieldShape = (
    type: Model,
    property: typeof type.properties extends Map<string, infer P> ? P : never,
    mode: GoPayloadMode,
  ) => {
    const options = { mode: mode === 'input' ? ('input' as const) : undefined }
    const mapped = goType(program, property.type, options)
    const optional =
      property.optional ||
      (mode === 'input' && property.defaultValue !== undefined)
    const pointerOptional =
      optional && !mapped.jsonNullable && (!mapped.nilable || mode === 'input')
    return {
      name: goExportedName(property.name),
      wireName: resolveEncodedName(program, property, 'application/json'),
      optional,
      pointerOptional,
    }
  }

  // Whether the two planned declarations render identical Go code up to the
  // declared type name, mirroring the GoModels render strategies.
  const declarationsMatch = (
    left: Type,
    leftMode: GoPayloadMode,
    right: Type,
    rightMode: GoPayloadMode,
    pending: Map<string, string>,
    inProgress: Set<string>,
  ): boolean => {
    if (left.kind === 'Model' && right.kind === 'Model') {
      const leftProperties = filteredProperties(left, leftMode)
      const rightProperties = filteredProperties(right, rightMode)
      if (leftProperties.length !== rightProperties.length) {
        return false
      }
      for (let index = 0; index < leftProperties.length; index++) {
        const leftProperty = leftProperties[index]!
        const rightProperty = rightProperties[index]!
        let leftShape
        let rightShape
        try {
          leftShape = fieldShape(left, leftProperty, leftMode)
          rightShape = fieldShape(right, rightProperty, rightMode)
        } catch {
          return false
        }
        if (
          leftShape.name !== rightShape.name ||
          leftShape.wireName !== rightShape.wireName ||
          leftShape.optional !== rightShape.optional ||
          leftShape.pointerOptional !== rightShape.pointerOptional ||
          !typesMatch(
            leftProperty.type,
            leftMode,
            rightProperty.type,
            rightMode,
            pending,
            inProgress,
          )
        ) {
          return false
        }
      }
      return true
    }

    if (left.kind === 'Union' && right.kind === 'Union') {
      return unionDeclarationsMatch(
        left,
        leftMode,
        right,
        rightMode,
        pending,
        inProgress,
      )
    }

    return false
  }

  const unionRenderStrategy = (
    union: Union,
  ): 'nullable' | 'strenum' | 'stringlike' | 'tagged' | 'alias' | 'invalid' => {
    if (nullableUnionElement(union)) {
      return 'nullable'
    }
    const variants = [...union.variants.values()]
    if (
      variants.length > 0 &&
      variants.every((variant) => variant.type.kind === 'String')
    ) {
      return 'strenum'
    }
    if (
      variants.length > 0 &&
      variants.every((variant) => isStringLike(program, variant.type))
    ) {
      return 'stringlike'
    }
    if (
      variants.length > 0 &&
      variants.every((variant) => variant.type.kind === 'Model')
    ) {
      return 'tagged'
    }
    const concrete = variants.filter(
      (variant) => variant.type.kind !== 'Intrinsic',
    )
    if (concrete.length === 1) {
      return 'alias'
    }
    return concrete.length > 1 ? 'tagged' : 'invalid'
  }

  const unionDeclarationsMatch = (
    left: Union,
    leftMode: GoPayloadMode,
    right: Union,
    rightMode: GoPayloadMode,
    pending: Map<string, string>,
    inProgress: Set<string>,
  ): boolean => {
    const strategy = unionRenderStrategy(left)
    if (strategy !== unionRenderStrategy(right) || strategy === 'invalid') {
      return false
    }

    const leftVariants = [...left.variants.values()]
    const rightVariants = [...right.variants.values()]

    switch (strategy) {
      case 'nullable':
        return typesMatch(
          nullableUnionElement(left)!,
          leftMode,
          nullableUnionElement(right)!,
          rightMode,
          pending,
          inProgress,
        )
      case 'strenum': {
        if (leftVariants.length !== rightVariants.length) {
          return false
        }
        return leftVariants.every((variant, index) => {
          const other = rightVariants[index]!
          return (
            variant.type.kind === 'String' &&
            other.type.kind === 'String' &&
            goExportedName(String(variant.name)) ===
              goExportedName(String(other.name)) &&
            variant.type.value === other.type.value
          )
        })
      }
      case 'stringlike':
        return true
      case 'alias': {
        const leftConcrete = leftVariants.filter(
          (variant) => variant.type.kind !== 'Intrinsic',
        )[0]!
        const rightConcrete = rightVariants.filter(
          (variant) => variant.type.kind !== 'Intrinsic',
        )[0]!
        return typesMatch(
          leftConcrete.type,
          leftMode,
          rightConcrete.type,
          rightMode,
          pending,
          inProgress,
        )
      }
      case 'tagged': {
        const leftConcrete = leftVariants.filter(
          (variant) => variant.type.kind !== 'Intrinsic',
        )
        const rightConcrete = rightVariants.filter(
          (variant) => variant.type.kind !== 'Intrinsic',
        )
        if (leftConcrete.length !== rightConcrete.length) {
          return false
        }
        const leftDiscriminator = discriminatorProperty(
          program,
          left,
          leftConcrete.flatMap((variant) =>
            variant.type.kind === 'Model' ? [variant.type] : [],
          ),
        )
        const rightDiscriminator = discriminatorProperty(
          program,
          right,
          rightConcrete.flatMap((variant) =>
            variant.type.kind === 'Model' ? [variant.type] : [],
          ),
        )
        if (leftDiscriminator?.wireName !== rightDiscriminator?.wireName) {
          return false
        }
        return leftConcrete.every((variant, index) => {
          const other = rightConcrete[index]!
          if (variant.type.kind !== other.type.kind) {
            return false
          }
          if (variant.type.kind === 'Model' && other.type.kind === 'Model') {
            if (
              discriminatorLiteral(variant.type, leftDiscriminator) !==
              discriminatorLiteral(other.type, rightDiscriminator)
            ) {
              return false
            }
          } else if (
            variantAccessorName(program, '', variant) !==
            variantAccessorName(program, '', other)
          ) {
            return false
          }
          return typesMatch(
            variant.type,
            leftMode,
            other.type,
            rightMode,
            pending,
            inProgress,
          )
        })
      }
    }
  }

  const sortedEntries = (): PlannedDeclaration[] =>
    [...plan]
      .flatMap(([type, declarations]) =>
        declarations.map((declaration) => ({ type, declaration })),
      )
      .sort((leftEntry, rightEntry) =>
        leftEntry.declaration.name < rightEntry.declaration.name
          ? -1
          : leftEntry.declaration.name > rightEntry.declaration.name
            ? 1
            : 0,
      )

  for (let changed = true; changed; ) {
    changed = false
    for (const { type, declaration } of sortedEntries()) {
      if (aliases.has(declaration.name)) {
        continue
      }
      const rest = projectionPrefixRest(declaration.name)
      if (rest === undefined) {
        continue
      }
      const declaredMatch = byDeclaredName.get(rest)
      const target =
        declarationsByName.get(resolveThrough(rest, new Map())) ??
        (declaredMatch === 'ambiguous' ? undefined : declaredMatch)
      if (!target || target.type === type) {
        continue
      }
      const targetName = resolveThrough(target.declaration.name, new Map())
      if (aliases.has(target.declaration.name)) {
        // The canonical itself collapsed; follow it to its final home.
        const followed = declarationsByName.get(targetName)
        if (!followed || followed.type === type) {
          continue
        }
      }

      const pending = new Map<string, string>()
      if (
        declarationsMatch(
          type,
          declaration.mode,
          target.type,
          target.declaration.mode,
          pending,
          new Set([`${declaration.name}->${targetName}`]),
        )
      ) {
        aliases.set(declaration.name, targetName)
        for (const [from, to] of pending) {
          if (!aliases.has(from)) {
            aliases.set(from, to)
          }
        }
        changed = true
      }
    }
  }

  // Flatten chains and drop the collapsed declarations from the plan.
  for (const [name] of aliases) {
    aliases.set(name, resolveThrough(name, new Map()))
  }
  for (const [type, declarations] of plan) {
    const kept = declarations.filter(
      (declaration) => !aliases.has(declaration.name),
    )
    if (kept.length === declarations.length) {
      continue
    }
    if (kept.length === 0) {
      plan.delete(type)
    } else {
      plan.set(type, kept)
    }
  }

  return aliases
}
