import "../shared/index.tsp";
import "./ratecard.tsp";

namespace ProductCatalog;

/**
 * The status of a plan.
 */
@friendlyName("BillingPlanStatus")
enum PlanStatus {
  draft: "draft",
  active: "active",
  archived: "archived",
  scheduled: "scheduled",
}

/**
 * References an exact plan.
 */
@friendlyName("BillingPlanReference")
model PlanReference {
  ...PickProperties<Shared.ResourceWithKey, "key" | "name" | "id">;

  /**
   * The plan version.
   */
  version: integer;
}

/**
 * Plans provide a template for subscriptions.
 */
@friendlyName("Plan")
model Plan {
  ...PickProperties<Shared.ResourceWithKey, "key" | "name" | "id">;

  /**
   * Version of the plan. Incremented when the plan is updated.
   */
  @summary("Version")
  @minValue(1)
  @visibility(Lifecycle.Read)
  version: integer = 1;

  /**
   * The currency code of the plan.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Currency")
  currency: Shared.CurrencyCode = "USD";

  /**
   * The default billing cadence for subscriptions using this plan.
   * Defines how often customers are billed using ISO8601 duration format.
   * Examples: "P1M" (monthly), "P3M" (quarterly), "P1Y" (annually).
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Billing cadence")
  @encode(DurationKnownEncoding.ISO8601)
  @example(duration.fromISO("P1M"))
  billingCadence: duration;
}
