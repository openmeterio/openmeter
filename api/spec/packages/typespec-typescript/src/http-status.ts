import type { HttpStatusCodesEntry } from '@typespec/http'

/**
 * Whether a response's status code(s) mark it as a success response — the ONE
 * definition every walk must share (operation schemas, request/response types,
 * visibility, casing gate). A divergent local copy is how an operation's
 * compile-time types and its runtime mapping drift apart: a body one walk sees
 * and another skips ships with unverified casing or a mismatched schema.
 *
 * Semantics: a single status is success in 200–299; a `{start, end}` range is
 * success when it overlaps 200–299; the `"*"` default-response wildcard IS
 * success — its body is mapped at runtime when no numbered success response
 * exists, so every compile-time walk (including the casing gate) must examine
 * it too.
 */
export function isSuccessStatus(statusCodes: HttpStatusCodesEntry): boolean {
  if (statusCodes === '*') {
    return true
  }
  if (typeof statusCodes === 'number') {
    return statusCodes >= 200 && statusCodes < 300
  }
  return statusCodes.start < 300 && statusCodes.end >= 200
}
