import "@typespec/http";
import "../types.tsp";
import "./ratecards.tsp";

namespace OpenMeter.ProductCatalog;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

/**
 * The status of the addon defined by the effectiveFrom and effectiveTo properties.
 */
@friendlyName("AddonStatus")
enum AddonStatus {
  draft: "draft",
  active: "active",
  archived: "archived",
}

/**
 * The instanceType of the addon.
 * Single instance addons can be added to subscription only once while addons with multiple type can be added more then once.
 */
@friendlyName("AddonInstanceType")
enum AddonInstanceType {
  single: "single",
  multiple: "multiple",
}

/**
 * Addon allows extending subscriptions with compatible plans with additional ratecards.
 */
@friendlyName("Addon")
model Addon {
  ...UniqueResource;

  /**
   * Set of key-value pairs managed by the system. Cannot be modified by user.
   */
  @visibility(Lifecycle.Read)
  @summary("Annotations")
  annotations?: Annotations;

  /**
   * Version of the addon. Incremented when the addon is updated.
   */
  @summary("Version")
  @minValue(1)
  @visibility(Lifecycle.Read)
  version: integer = 1;

  /**
   * The instanceType of the addons. Can be "single" or "multiple".
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("InstanceType")
  instanceType: AddonInstanceType;

  /**
   * The currency code of the addon.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Currency")
  currency: CurrencyCode = "USD";

  /**
   * The date and time when the addon becomes effective. When not specified, the addon is a draft.
   */
  @visibility(Lifecycle.Read)
  @summary("Effective start date")
  effectiveFrom?: DateTime;

  /**
   * The date and time when the addon is no longer effective. When not specified, the addon is effective indefinitely.
   */
  @visibility(Lifecycle.Read)
  @summary("Effective end date")
  effectiveTo?: DateTime;

  /**
   * The status of the addon.
   * Computed based on the effective start and end dates:
   *  - draft = no effectiveFrom
   *  - active = effectiveFrom <= now < effectiveTo
   *  - archived  = effectiveTo <= now
   */
  @summary("Status")
  @visibility(Lifecycle.Read)
  status: AddonStatus;

  /**
   * The rate cards of the addon.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Rate cards")
  rateCards: AddonRateCard[];
}

/**
 * Order by options for addons.
 */
@friendlyName("AddonOrderBy")
enum AddonOrderBy {
  id: "id",
  key: "key",
  version: "version",
  createdAt: "created_at",
  updatedAt: "updated_at",
}

/**
 * The addon mode defines how the ratecards included in the addon are applied in a subscription.
 * Override mode means that the ratecards in addon take precedence over the ones which are defined in a plan with for the same feature.
 * While the extend mode extends the ratecards from plan for the same feature.
 */
@friendlyName("AddonRateCardMode")
enum AddonRateCardMode {
  override: "override",
  extend: "extend",
}

/**
 * AddonRateCard defines the ratecard itself and how it needs to be applied to plan/subscription.
 */
@friendlyName("AddonRateCard")
model AddonRateCard {
  /*
   * The ratecard definition.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  rateCard: RateCard;

  /*
   * The mode of the addon ratecard. Can be "override" or "extend".
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  mode: AddonRateCardMode;
}
