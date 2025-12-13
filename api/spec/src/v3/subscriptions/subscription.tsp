import "../shared/index.tsp";

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
 * Subscription create request.
 */
@friendlyName("BillingSubscriptionCreate")
model SubscriptionCreate {
  ...Shared.CreateRequest<OmitProperties<Subscription, "billing_anchor">>;

  /**
   * The billing anchor of the subscription.
   * Defaults to now if not provided.
   */
  @visibility(Lifecycle.Create)
  @summary("Billing anchor")
  billing_anchor?: Shared.DateTime;
}

/**
 * Subscription.
 */
@friendlyName("BillingSubscription")
model Subscription {
  ...OmitProperties<Shared.Resource, "name" | "description">;

  /**
   * The customer ID of the subscription.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Customer ID")
  customer_id: Shared.ULID;

  /**
   * The plan ID of the subscription.
   * Set if subscription is created from a plan.
   *
   * ID or Key of the plan is required if creating a subscription from a plan.
   * If both are provided, the ID will be used.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Plan ID")
  plan_id?: Shared.ULID;

  /**
   * The plan Key of the subscription, if any.
   * Set if subscription is created from a plan.
   *
   * ID or Key of the plan is required if creating a subscription from a plan.
   * If both are provided, the ID will be used.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Plan Key")
  plan_key?: Shared.ResourceKey;

  /**
   * The plan version of the subscription, if any.
   * Set if subscription is created from a plan.
   *
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Plan Version")
  plan_version?: integer;

  /**
   * The billing anchor of the subscription.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Billing anchor")
  billing_anchor: Shared.DateTime;

  /**
   * The status of the subscription.
   */
  @visibility(Lifecycle.Read)
  @summary("Status")
  status: SubscriptionStatus;
}
