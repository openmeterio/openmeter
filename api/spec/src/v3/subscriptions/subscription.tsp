import "@typespec/openapi";
import "@typespec/openapi3";
import "../shared/index.tsp";

using TypeSpec.OpenAPI;

namespace Subscriptions;

/**
 * Reference to a plan by its id or key and version.
 */
@oneOf
@friendlyName("PlanReference")
union PlanReference {
  id: Shared.ULID,
  keyVersion: {
    key: Shared.ResourceKey,
    version?: integer,
  },
}

/**
 * A subscription is a customer's subscription to a plan.
 */
@friendlyName("BillingSubscription")
model Subscription {
  ...Shared.Resource;

  /**
   * Plan reference.
   */
  plan: PlanReference;
}
