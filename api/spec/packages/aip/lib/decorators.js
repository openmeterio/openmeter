import { isNullType } from '@typespec/compiler'
import { $ } from '@typespec/compiler/typekit'

// Rewrites every optional, not-already-nullable property type to `T | null` so
// upsert request templates can express tri-state fields (omit = keep, null =
// unset, value = replace) without hand-written per-model unions. Only safe on
// property copies (`model ... is ...` templates); never apply to a shared read
// model directly.
export function $withNullableOptionalProperties(context, target) {
  const tk = $(context.program)

  for (const property of target.properties.values()) {
    if (!property.optional) {
      continue
    }

    const type = property.type
    if (isNullType(type)) {
      continue
    }

    if (
      type.kind === 'Union' &&
      [...type.variants.values()].some((variant) => isNullType(variant.type))
    ) {
      continue
    }

    property.type = tk.union.create([type, tk.intrinsic.null])
  }
}

export const $decorators = {
  Shared: {
    withNullableOptionalProperties: $withNullableOptionalProperties,
  },
}
