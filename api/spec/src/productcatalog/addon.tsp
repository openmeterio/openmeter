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
 * The type of the addon.
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
  rateCards: RateCard[];
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
