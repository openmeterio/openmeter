import "@typespec/http";
import "../types.tsp";
import "./ratecards.tsp";

namespace OpenMeter.ProductCatalog;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

/**
 * A subscription addon, a concrete instance of an addon.
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
  rateCards: RateCard[];
}
