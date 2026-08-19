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
   * The display name of the subscription.
   *
   * Only populated where the referencing endpoint documents a `subscription` expand
   * that resolves it.
   */
  @visibility(Lifecycle.Read)
  @summary("Subscription name")
  name?: string;

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
