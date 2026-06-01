import "../productcatalog/addon.tsp";

namespace Subscriptions;

/**
 * Addon purchased with a subscription.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "active_from and active_to should not be grouped"
@friendlyName("SubscriptionAddon")
model SubscriptionAddon {
  ...OmitProperties<Shared.Resource, "name" | "description">;

  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  @visibility(Lifecycle.Read)
  @minLength(1)
  @maxLength(256)
  name: string;

  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  @maxLength(1024)
  @visibility(Lifecycle.Read)
  description?: string;

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

  /**
   * The timing of the operation. After the create or update, a new entry will be
   * created in the timeline.
   */
  @summary("Timing")
  @visibility(Lifecycle.Create, Lifecycle.Update)
  timing: SubscriptionEditTiming;

  // TODO consider adding rate cards
}
