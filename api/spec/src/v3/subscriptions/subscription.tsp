import "../shared/index.tsp";
import "../packages/plan.tsp";

namespace Subscriptions;

/**
 * Subscription status.
 */
@friendlyName("BillingSubscriptionStatus")
enum SubscriptionStatus {
  active,
  inactive,
  canceled,
  scheduled,
}

/**
 * Customer subscription.
 */
@friendlyName("BillingSubscription")
model Subscription {
  ...OmitProperties<Shared.Resource, "name" | "description">;

  /**
   * The plan ID of the subscription.
   */
  @visibility(Lifecycle.Read)
  @summary("Plan ID")
  plan_id: Shared.ULID;

  /**
   * The customer ID of the subscription.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Customer ID")
  customer_id: Shared.ULID;

  /**
   * The plan reference for the subscription.
   */
  @visibility(Lifecycle.Create, Lifecycle.Update)
  @summary("Plan Reference")
  plan: Packages.Reference;

  /**
   * The billing anchor of the subscription.
   */
  @visibility(Lifecycle.Read)
  @summary("Billing anchor")
  billing_anchor: Shared.DateTime;

  /**
   * The billing cadence for the subscriptions.
   * Defines how often customers are billed using ISO8601 duration format.
   * Examples: "P1M" (monthly), "P3M" (quarterly), "P1Y" (annually).
   */
  @visibility(Lifecycle.Read)
  @summary("Billing cadence")
  @encode(DurationKnownEncoding.ISO8601)
  @example(duration.fromISO("P1M"))
  billing_cadence: duration;

  /**
   * The status of the subscription.
   */
  @visibility(Lifecycle.Read)
  @summary("Status")
  status: SubscriptionStatus;
}
