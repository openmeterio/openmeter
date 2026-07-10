import {
  getFriendlyName,
  type Operation,
  type Program,
} from '@typespec/compiler'
import { getOperationId } from '@typespec/openapi'

const NOISE_TOKENS = new Set(['metering'])
const ACRONYMS = new Map([
  ['csv', 'CSV'],
  ['json', 'JSON'],
])
const SPLIT_BY_INTERFACE = new Set(['ProductCatalog'])

function lowerFirst(name: string): string {
  const match = name.match(/^([A-Z]{2,})([A-Z][a-z].*)$/)
  if (match?.[1] && match[2]) {
    return match[1].toLowerCase() + match[2]
  }

  return name.charAt(0).toLowerCase() + name.slice(1)
}

export function resourceWords(resource: string): string[] {
  return resource
    .replace(/([A-Z]+)([A-Z][a-z])/g, '$1 $2')
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .split(/[\s\-_]+/)
    .filter(Boolean)
    .map((word) => word.toLowerCase())
}

function resourceTokens(
  resource: string,
  nestPath: string[] = [],
): Set<string> {
  const tokens = new Set(NOISE_TOKENS)
  for (const word of [
    ...resourceWords(resource),
    ...nestPath.flatMap(resourceWords),
  ]) {
    tokens.add(word)
    tokens.add(word.endsWith('s') ? word.slice(0, -1) : `${word}s`)
  }

  return tokens
}

export function methodNameOf(
  program: Program,
  operation: Operation,
  resource: string,
  nestPath: string[] = [],
): string {
  const operationID =
    getFriendlyName(program, operation) ??
    getOperationId(program, operation) ??
    operation.name
  const strip = resourceTokens(resource, nestPath)
  const kept = resourceWords(operationID).filter((part) => !strip.has(part))
  const parts = kept.length > 0 ? kept : resourceWords(operationID)

  return exportedMethodName(parts)
}

export function methodNameFromOperationName(operationName: string): string {
  return exportedMethodName(resourceWords(operationName))
}

function exportedMethodName(parts: string[]): string {
  return parts
    .map(
      (part) =>
        ACRONYMS.get(part) ?? part.charAt(0).toUpperCase() + part.slice(1),
    )
    .join('')
}

export function sourceOf(operation: Operation): {
  chain: string[]
  interface?: string
} {
  const source =
    operation.interface?.sourceInterfaces?.[0] ??
    operation.sourceOperation?.interface
  const chain: string[] = []

  for (
    let namespace = source?.namespace;
    namespace?.name;
    namespace = namespace.namespace
  ) {
    chain.unshift(namespace.name)
  }

  return { chain, interface: source?.name }
}

export function operationNestPath(
  operation: Operation,
  resource: string,
): string[] {
  const { chain } = sourceOf(operation)
  if (SPLIT_BY_INTERFACE.has(chain[0] ?? '') || chain[0] !== resource) {
    return []
  }

  return chain.slice(1)
}

function interfaceResource(interfaceName: string): string {
  return pluralize(interfaceName.replace(/Operations$/, ''))
}

export function groupOperations(
  operations: Operation[],
): Map<string, Operation[]> {
  const groups = new Map<string, Operation[]>()

  for (const operation of operations) {
    const { chain, interface: sourceInterface } = sourceOf(operation)
    const top = chain[0]
    if (!top) {
      const qualifiedName = operation.interface
        ? `${operation.interface.name}.${operation.name}`
        : operation.name
      throw new Error(
        `typespec-go: cannot place operation ${qualifiedName} in a resource group: its source declaration is not inside a named namespace. Declare it in an interface that extends a resource namespace interface (for example \`interface Endpoints extends Customers.Operations\`) or reference a namespaced operation with \`is\`, so the emitter knows which Go sub-client owns it.`,
      )
    }

    const resource =
      SPLIT_BY_INTERFACE.has(top) && sourceInterface
        ? interfaceResource(sourceInterface)
        : top
    const existing = groups.get(resource)
    if (existing) {
      existing.push(operation)
    } else {
      groups.set(resource, [operation])
    }
  }

  return groups
}

export function pluralize(word: string): string {
  if (word.endsWith('s')) {
    return word
  }
  if (/(x|z|ch|sh)$/i.test(word)) {
    return `${word}es`
  }
  if (/[^aeiou]y$/i.test(word)) {
    return `${word.slice(0, -1)}ies`
  }

  return `${word}s`
}

export function namespaceNames(resource: string): {
  class: string
  getter: string
} {
  const className = resource.charAt(0).toUpperCase() + resource.slice(1)
  return { class: className, getter: lowerFirst(className) }
}
