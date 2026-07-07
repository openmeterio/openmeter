export function toCamelCase(name: string): string {
  return name.replace(/_([a-z0-9])/g, (_m, c: string) => c.toUpperCase())
}

export function toSnakeCase(name: string): string {
  return name.replace(/([A-Z])/g, (_m, c: string) => `_${c.toLowerCase()}`)
}

/**
 * True when a name survives the camel→snake→camel round-trip, i.e. its wire form
 * is recoverable from its public (camelized) form by {@link toSnakeCase} alone.
 * The boundary mapper relies on this for every key it does not carry an explicit
 * wire name for; the codegen gate asserts it over every emitted wire key so a
 * non-derivable name fails the build instead of silently shipping a wrong key.
 */
export function isCasingDerivable(wireName: string): boolean {
  return toSnakeCase(toCamelCase(wireName)) === wireName
}
