import "../productcatalog/addon.tsp";
import "../productcatalog/ratecard.tsp";

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

  /**
   * The timeline of the add-on. The returned periods are sorted and continuous.
   */
  @visibility(Lifecycle.Read)
  @summary("Timeline")
  @example(#[
    #{
      quantity: 1,
      active_from: Shared.DateTime.fromISO("2025-01-01T00:00:00Z"),
      active_to: Shared.DateTime.fromISO("2025-01-02T00:00:00Z"),
    },
    #{
      quantity: 0,
      active_from: Shared.DateTime.fromISO("2025-01-02T00:00:00Z"),
      active_to: Shared.DateTime.fromISO("2025-01-03T00:00:00Z"),
    },
    #{
      quantity: 1,
      active_from: Shared.DateTime.fromISO("2025-01-03T00:00:00Z"),
    }
  ])
  timeline: SubscriptionAddonTimelineSegment[];

  /**
   * The rate cards of the add-on.
   */
  @visibility(Lifecycle.Read)
  @summary("Rate cards")
  rate_cards: SubscriptionAddonRateCard[];
}

/**
 * A rate card for a subscription add-on.
 */
@friendlyName("SubscriptionAddonRateCard")
model SubscriptionAddonRateCard {
  /**
   * The rate card.
   */
  @summary("Rate card")
  rate_card: ProductCatalog.RateCard;

  /**
   * The IDs of the subscription items that this rate card belongs to.
   */
  @summary("Affected subscription item IDs")
  @visibility(Lifecycle.Read)
  affected_subscription_item_ids: Shared.ULID[];
}

/**
 * A subscription add-on event.
 */
@friendlyName("SubscriptionAddonTimelineSegment")
model SubscriptionAddonTimelineSegment {
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
   * The quantity of the add-on for the given period.
   */
  @summary("Quantity")
  @visibility(Lifecycle.Read)
  @example(1)
  @minValue(0)
  quantity: integer;
}
