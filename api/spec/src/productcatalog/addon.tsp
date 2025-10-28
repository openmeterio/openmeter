import "@typespec/http";
import "../types.tsp";
import "./ratecards.tsp";

namespace OpenMeter;

/**
 * The status of the add-on defined by the effectiveFrom and effectiveTo properties.
 */
@friendlyName("AddonStatus")
enum AddonStatus {
  draft: "draft",
  active: "active",
  archived: "archived",
}

/**
 * The instanceType of the add-on.
 * Single instance add-ons can be added to subscription only once while add-ons with multiple type can be added more then once.
 */
@friendlyName("AddonInstanceType")
enum AddonInstanceType {
  single: "single",
  multiple: "multiple",
}

/**
 * Add-on allows extending subscriptions with compatible plans with additional ratecards.
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
   * Version of the add-on. Incremented when the add-on is updated.
   */
  @summary("Version")
  @minValue(1)
  @visibility(Lifecycle.Read)
  version: integer = 1;

  /**
   * The instanceType of the add-ons. Can be "single" or "multiple".
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("InstanceType")
  instanceType: AddonInstanceType;

  /**
   * The currency code of the add-on.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Currency")
  currency: CurrencyCode = "USD";

  /**
   * The date and time when the add-on becomes effective. When not specified, the add-on is a draft.
   */
  @visibility(Lifecycle.Read)
  @summary("Effective start date")
  effectiveFrom?: DateTime;

  /**
   * The date and time when the add-on is no longer effective. When not specified, the add-on is effective indefinitely.
   */
  @visibility(Lifecycle.Read)
  @summary("Effective end date")
  effectiveTo?: DateTime;

  /**
   * The status of the add-on.
   * Computed based on the effective start and end dates:
   *  - draft = no effectiveFrom
   *  - active = effectiveFrom <= now < effectiveTo
   *  - archived  = effectiveTo <= now
   */
  @summary("Status")
  @visibility(Lifecycle.Read)
  status: AddonStatus;

  /**
   * The rate cards of the add-on.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Rate cards")
  rateCards: RateCard[];

  /**
   * List of validation errors.
   */
  @visibility(Lifecycle.Read)
  @summary("Validation errors")
  validationErrors: ValidationError[] | null;
}

/**
 * Order by options for add-ons.
 */
@friendlyName("AddonOrderBy")
enum AddonOrderBy {
  id: "id",
  key: "key",
  version: "version",
  createdAt: "created_at",
  updatedAt: "updated_at",
}
