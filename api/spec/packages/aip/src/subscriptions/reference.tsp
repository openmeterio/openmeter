import "../shared/index.tsp";

namespace Subscriptions;

/**
 * Subscription reference represents a reference to the specific subscription item
 * this entity represents.
 */
@friendlyName("BillingSubscriptionReference")
model SubscriptionReference {
  /**
   * The ID of the subscription.
   */
  @visibility(Lifecycle.Read)
  @summary("Subscription ID")
  id: Shared.ULID;

  /**
   * The phase of the subscription.
   */
  @visibility(Lifecycle.Read)
  @summary("Phase ID")
  phase: {
    /**
     * The ID of the phase.
     */
    @visibility(Lifecycle.Read)
    @summary("Phase ID")
    id: Shared.ULID;

    /**
     * The item of the phase.
     */
    @visibility(Lifecycle.Read)
    @summary("Item ID")
    item: {
      /**
       * The ID of the item.
       */
      @visibility(Lifecycle.Read)
      @summary("Item ID")
      id: Shared.ULID;
    };
  };
}
