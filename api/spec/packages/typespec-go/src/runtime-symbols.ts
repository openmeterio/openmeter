import type { Type } from '@typespec/compiler'

// Exported names owned by the static runtime templates and fixed client
// scaffolding. TypeSpec-generated declarations share Go's package-level
// namespace with these symbols, so any accidental overlap must stop emission
// before the generated SDK reaches go build.
export const RESERVED_GO_SYMBOL_NAMES = new Set([
  'APIError',
  'AsAPIError',
  'Bool',
  'BooleanFilter',
  'Client',
  'CursorPageParams',
  'DateTimeFilter',
  'DecodeAPIError',
  'ErrEmptyID',
  'Int',
  'Many',
  'New',
  'Null',
  'Nullable',
  'NullableValue',
  'Numeric',
  'NumericFilter',
  'One',
  'OneOrMany',
  'Option',
  'PageMeta',
  'PageParams',
  'PaginatedMeta',
  'Ptr',
  'Sort',
  'SortOrder',
  'SortOrderAsc',
  'SortOrderDesc',
  'String',
  'StringExactFilter',
  'StringFilter',
  'Time',
  'Version',
  'WithHTTPClient',
  'WithToken',
  'WithUserAgent',
])

// TypeSpec constructs that intentionally map to runtime-owned SDK shapes rather
// than generated model declarations.
const RUNTIME_BACKED_TYPE_NAMES = new Set([
  'Numeric',
  'PageMeta',
  'PageParams',
  'PaginatedMeta',
  'SortQuery',
  'StringFilter',
  'StringFieldFilter',
  'StringFieldFilterExact',
])

export function isRuntimeBackedTypeName(
  name: string,
  kind: Type['kind'],
): boolean {
  if (name === 'String') {
    return kind === 'Scalar'
  }

  return (
    RUNTIME_BACKED_TYPE_NAMES.has(name) ||
    name.endsWith('FieldFilter') ||
    name.endsWith('FieldFilterExact')
  )
}

export function conflictsWithReservedGoSymbol(
  name: string,
  kind: Type['kind'],
): boolean {
  return (
    RESERVED_GO_SYMBOL_NAMES.has(name) && !isRuntimeBackedTypeName(name, kind)
  )
}
