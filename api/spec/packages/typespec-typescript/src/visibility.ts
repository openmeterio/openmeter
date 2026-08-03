import {
  type Model,
  type ModelProperty,
  type Operation,
  type Program,
  type Type,
} from '@typespec/compiler'
import { $ } from '@typespec/compiler/typekit'
import { isVisible, Visibility } from '@typespec/http'
import { isSuccessStatus } from './http-status.js'

/**
 * Models reachable from an operation response body, keyed by program. A model in
 * this set is emitted in a read context, so its `@visibility(Lifecycle.Create)`-
 * /`Update`-only properties (which the server never returns) must be dropped from
 * the interface and zod schema.
 *
 * Request-only models are deliberately absent: the spec emits read and request
 * payloads as distinct model trees (`CreditGrant` vs `CreateCreditGrantRequest`),
 * so a request body such as `SubscriptionCreate` — which declares a Create-only
 * `billing_anchor` directly — keeps every property. A global per-property Read
 * filter would wrongly strip those create fields; gating on response-reachability
 * is what makes the filter safe.
 */
const responseReachableByProgram = new WeakMap<Program, Set<Model>>()

/**
 * Record which models are reachable from a response body, so {@link isReadVisible}
 * can gate visibility filtering on read context. Call once per emit, before any
 * type is walked.
 */
export function setResponseReachableModels(
  program: Program,
  models: Set<Model>,
): void {
  responseReachableByProgram.set(program, models)
}

/**
 * Whether a property survives into a model's emitted (read) shape.
 *
 * A property is dropped only when its model is response-reachable AND the
 * property is not visible in `Lifecycle.Read` — i.e. a create-/update-only field
 * leaking into a response type. For request-only models (not response-reachable)
 * every property is kept, because their create-only fields are legitimate request
 * input. Without the response-reachable holder set (e.g. in unit tests that don't
 * compute it) nothing is filtered, preserving prior behavior.
 */
export function isReadVisible(
  program: Program,
  model: Model,
  prop: ModelProperty,
): boolean {
  const reachable = responseReachableByProgram.get(program)
  if (!reachable || !reachable.has(model)) {
    return true
  }
  return isVisible(program, prop, Visibility.Read)
}

/**
 * The set of models transitively reachable from the success-response body of any
 * operation. Descends through properties, base models, array/record element
 * types, union variants, and tuples — the same shape the schema/interface walkers
 * traverse — so a create-only field nested anywhere under a response is caught.
 */
export function computeResponseReachableModels(
  program: Program,
  operations: Operation[],
): Set<Model> {
  const tk = $(program)
  const reachable = new Set<Model>()

  const visit = (type: Type | undefined): void => {
    if (!type) {
      return
    }
    switch (type.kind) {
      case 'Model': {
        if (reachable.has(type)) {
          return
        }
        reachable.add(type)
        if (type.indexer) {
          visit(type.indexer.value)
        }
        if (type.baseModel) {
          visit(type.baseModel)
        }
        // Descend into subclasses too: a model-form `@discriminator` hierarchy
        // returned as a response is reached through its base, and each concrete
        // subtype carries its own read-side properties to filter.
        for (const derived of type.derivedModels) {
          visit(derived)
        }
        for (const prop of type.properties.values()) {
          visit(prop.type)
        }
        break
      }
      case 'Union':
        for (const variant of type.variants.values()) {
          visit(variant.type)
        }
        break
      case 'Tuple':
        for (const value of type.values) {
          visit(value)
        }
        break
      default:
        break
    }
  }

  for (const op of operations) {
    const httpOp = tk.httpOperation.get(op)
    for (const response of httpOp.responses) {
      // Only success bodies define the read shape; error envelopes carry their
      // own models and would otherwise pull unrelated types into the read set.
      if (!isSuccessStatus(response.statusCodes)) {
        continue
      }
      for (const content of response.responses) {
        visit(content.body?.type)
      }
    }
  }

  return reachable
}
