/**
 * Strip a single configured prefix from `name`, anchored at the start, if any
 * applies. The match requires the character after the prefix to start a new
 * PascalCase word (an uppercase letter).
 */
export function stripOnePrefix(
  name: string,
  prefixes: readonly string[],
): string {
  for (const prefix of prefixes) {
    if (
      prefix &&
      name.length > prefix.length &&
      name.startsWith(prefix) &&
      /[A-Z]/.test(name[prefix.length]!)
    ) {
      return name.slice(prefix.length)
    }
  }
  return name
}

/**
 * Build a map from each input name to its resolved (possibly stripped) name. A
 * stripped candidate is adopted only when it is unique: it must not equal any
 * other name's original form, nor any other name's stripped candidate.
 */
export function resolveStrippedNames(
  names: Iterable<string>,
  prefixes: readonly string[],
): Map<string, string> {
  const all = [...names]
  const resolved = new Map<string, string>()

  if (prefixes.length === 0) {
    for (const name of all) resolved.set(name, name)
    return resolved
  }

  const originals = new Set(all)
  const candidate = new Map<string, string>()
  for (const name of all) {
    candidate.set(name, stripOnePrefix(name, prefixes))
  }

  const candidateCounts = new Map<string, number>()
  for (const target of candidate.values()) {
    candidateCounts.set(target, (candidateCounts.get(target) ?? 0) + 1)
  }

  for (const name of all) {
    const target = candidate.get(name)!
    if (target === name) {
      resolved.set(name, name)
      continue
    }
    const collides =
      originals.has(target) || (candidateCounts.get(target) ?? 0) > 1
    resolved.set(name, collides ? name : target)
  }

  return resolved
}
