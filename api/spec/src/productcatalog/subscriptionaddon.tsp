import "@typespec/http";
import "../types.tsp";
import "./ratecards.tsp";

namespace OpenMeter;

/**
 * A subscription add-on, represents concrete instances of an add-on for a given subscription.
 */
@friendlyName("SubscriptionAddon")
model SubscriptionAddon {
  ...Resource;

  /**
   * The cadence start of the resource.
   */
  @visibility(Lifecycle.Read)
  activeFrom: DateTime;

  /**
   * The cadence end of the resource.
   */
  @visibility(Lifecycle.Read)
  activeTo?: DateTime;

  /**
   * Partially populated add-on properties.
   */
  @summary("Addon")
  @visibility(Lifecycle.Read, Lifecycle.Create)
  addon: {
    /**
     * The ID of the add-on.
     */
    @summary("ID")
    @visibility(Lifecycle.Read, Lifecycle.Create)
    id: ULID;

    /**
     * A semi-unique identifier for the resource.
     */
    @visibility(Lifecycle.Read)
    @summary("Key")
    key: Key;

    /**
     * The version of the Add-on which templates this instance.
     */
    @summary("Version")
    @minValue(1)
    @visibility(Lifecycle.Read)
    version: integer = 1;

    /**
     * The instance type of the add-on.
     */
    @visibility(Lifecycle.Read)
    @summary("InstanceType")
    instanceType: AddonInstanceType;
  };

  /**
   * For which point in time the quantity was resolved to.
   */
  @summary("QuantityAt")
  @visibility(Lifecycle.Read)
  @example(DateTime.fromISO("2025-01-05T00:00:00Z"))
  quantityAt: DateTime;

  /**
   * The quantity of the add-on. Always 1 for single instance add-ons.
   */
  @summary("Quantity")
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @example(1)
  @minValue(0)
  quantity: integer;

  /**
   * The timing of the operation. After the create or update, a new entry will be created in the timeline.
   */
  @summary("Timing")
  @visibility(Lifecycle.Create, Lifecycle.Update)
  timing: SubscriptionTiming;

  /**
   * The timeline of the add-on. The returned periods are sorted and continuous.
   */
  @visibility(Lifecycle.Read)
  @summary("Timeline")
  @example(#[
    #{
      quantity: 1,
      activeFrom: DateTime.fromISO("2025-01-01T00:00:00Z"),
      activeTo: DateTime.fromISO("2025-01-02T00:00:00Z"),
    },
    #{
      quantity: 0,
      activeFrom: DateTime.fromISO("2025-01-02T00:00:00Z"),
      activeTo: DateTime.fromISO("2025-01-03T00:00:00Z"),
    },
    #{ quantity: 1, activeFrom: DateTime.fromISO("2025-01-03T00:00:00Z") }
  ])
  timeline: SubscriptionAddonTimelineSegment[];

  /**
   * The ID of the subscription.
   */
  @summary("SubscriptionID")
  @visibility(Lifecycle.Read)
  subscriptionId: ULID;

  /**
   * The rate cards of the add-on.
   */
  @visibility(Lifecycle.Read)
  @summary("Rate cards")
  rateCards: SubscriptionAddonRateCard[];
}

/**
 * A subscription add-on create body.
 */
@friendlyName("SubscriptionAddonCreate")
@withVisibility(Lifecycle.Create)
model SubscriptionAddonCreate {
  ...OmitProperties<
    DefaultKeyVisibility<SubscriptionAddon, Lifecycle.Read>,
    "addon"
  >;

  /**
   * The add-on to create.
   */
  @summary("Addon")
  @visibility(Lifecycle.Create)
  addon: {
    /**
     * The ID of the add-on.
     */
    id: ULID;
  };
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
  rateCard: RateCard;

  /**
   * The IDs of the subscription items that this rate card belongs to.
   */
  @summary("Affected subscription item IDs")
  @visibility(Lifecycle.Read)
  affectedSubscriptionItemIds: ULID[];
}

/**
 * A subscription add-on event.
 */
@friendlyName("SubscriptionAddonTimelineSegment")
model SubscriptionAddonTimelineSegment {
  ...global.CadencedResource;

  /**
   * The quantity of the add-on for the given period.
   */
  @summary("Quantity")
  @visibility(Lifecycle.Read)
  @example(1)
  @minValue(0)
  quantity: integer;
}
