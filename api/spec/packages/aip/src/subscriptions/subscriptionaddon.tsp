import "../productcatalog/addon.tsp";

namespace Subscriptions;

/**
 * SubscriptionAddon represents an addon associated with a subscription.
 */
@friendlyName("SubscriptionAddon")
model SubscriptionAddon {
  ...Shared.Resource;

  /**
   * The add-on associated with the subscription.
   */
  @summary("Add-on")
  @visibility(Lifecycle.Read, Lifecycle.Create)
  addon: Shared.ResourceReference<ProductCatalog.Addon>;

  /**
   * The quantity of the add-on. Always 1 for single instance add-ons.
   */
  @summary("Quantity")
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @minValue(1)
  quantity: integer;

  /**
   * An ISO-8601 timestamp representation of which point in time the quantity was
   * resolved to.
   */
  @visibility(Lifecycle.Read)
  quantity_at: Shared.DateTime;

  /**
   * An ISO-8601 timestamp representation of the cadence start of the resource.
   */
  @visibility(Lifecycle.Read)
  active_from: Shared.DateTime;

  /**
   * An ISO-8601 timestamp representation of the cadence end of the resource.
   */
  @visibility(Lifecycle.Read)
  active_to?: Shared.DateTime;
}
