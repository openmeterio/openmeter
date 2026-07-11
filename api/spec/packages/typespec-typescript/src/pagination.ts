import type {
  IndeterminateEntity,
  Model,
  Operation,
  Program,
  Type,
  Value,
} from '@typespec/compiler'
import {
  successResponseEnvelope,
  type ResolveInterface,
} from './sdk-operations.js'

export interface PaginationInfo {
  style: 'page' | 'cursor'
  /**
   * The documented interface name of a single list item (`Meter`, not
   * `MeterPagePaginatedResponse`) — what the emitted `<method>All` companion
   * yields.
   */
  itemInterface: string
}

export interface PaginationTemplates {
  page?: Model
  cursor?: Model
}

/**
 * The two shared response envelope templates pagination detection matches
 * against (`Shared.PagePaginatedResponse<T>` and
 * `Shared.CursorPaginatedResponse<T>` in `shared/responses.tsp`), resolved
 * once per emit. `Shared` is looked up by name because TypeSpec has no other
 * way to name "the two templates this emitter builds pagination around" —
 * but the per-operation match itself ({@link paginationInfo}) is by AST node
 * identity, not by name, so a `@friendlyName` rename or an interpolated name
 * that happens to collide can never produce a false match.
 */
export function findPaginationTemplates(program: Program): PaginationTemplates {
  const shared = program.getGlobalNamespaceType().namespaces.get('Shared')
  return {
    page: shared?.models.get('PagePaginatedResponse'),
    cursor: shared?.models.get('CursorPaginatedResponse'),
  }
}

function asType(
  entity: Type | Value | IndeterminateEntity | undefined,
): Type | undefined {
  return entity?.entityKind === 'Type' ? entity : undefined
}

/**
 * Whether `op`'s success response is page- or cursor-paginated and, when it
 * is, the item type a caller iterates. A response matches a style when its
 * envelope Model was instantiated from the corresponding template — every
 * instantiation of a TypeSpec generic model shares the declaration's syntax
 * node, so comparing `.node` identity is exact regardless of the
 * instantiation's own (possibly `@friendlyName`-interpolated) name.
 *
 * Returns undefined for a non-paginated response, or for a paginated one
 * whose item type has no documented interface to name in a companion's
 * `AsyncIterable<…>` — those operations get no `<method>All` companion (see
 * `sdk-files.ts`), per the emitter's "no name-string heuristics" contract:
 * an item type the SDK cannot already reference by name is not something a
 * companion can safely promise to yield.
 */
export function paginationInfo(
  program: Program,
  op: Operation,
  templates: PaginationTemplates,
  resolveInterface: ResolveInterface,
): PaginationInfo | undefined {
  const envelope = successResponseEnvelope(program, op)
  if (!envelope || envelope.kind !== 'Model') {
    return undefined
  }
  const style =
    templates.page && envelope.node === templates.page.node
      ? 'page'
      : templates.cursor && envelope.node === templates.cursor.node
        ? 'cursor'
        : undefined
  if (!style) {
    return undefined
  }
  const itemType = asType(envelope.templateMapper?.args?.[0])
  const itemInterface = resolveInterface(itemType)
  return itemInterface ? { style, itemInterface } : undefined
}
