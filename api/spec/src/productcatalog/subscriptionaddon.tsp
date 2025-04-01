import "@typespec/http";
import "../types.tsp";
import "./ratecards.tsp";

namespace OpenMeter.ProductCatalog;

/**
 * A subscription addon, a concrete instance of an addon for a given subscription.
 */
@friendlyName("SubscriptionAddon")
model SubscriptionAddon {
  ...UniqueResource;
  ...global.CadencedResource;

  /**
   * The version of the Addon which templates this instance.
   */
  @summary("Version")
  @minValue(1)
  @visibility(Lifecycle.Read)
  version: integer = 1;

  /**
   * The ID of the addon.
   */
  @summary("AddonID")
  @visibility(Lifecycle.Read)
  addonId: ULID;

  /**
   * The quantity of the addon. Always 1 for single instance add-ons.
   */
  @summary("Quantity")
  @visibility(Lifecycle.Read, Lifecycle.Create)
  quantity: integer;

  /**
   * The instanceType of the addon. Can be "single" or "multiple".
   */
  @visibility(Lifecycle.Read)
  @summary("InstanceType")
  instanceType: AddonInstanceType;

  /**
   * The ID of the subscription.
   */
  @summary("SubscriptionID")
  @visibility(Lifecycle.Read)
  subscriptionId: ULID;

  /**
   * The rate cards of the addon.
   */
  @visibility(Lifecycle.Read)
  @summary("Rate cards")
  rateCards: SubscriptionAddonRateCard[];
}

/**
 * A rate card for a subscription addon.
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
