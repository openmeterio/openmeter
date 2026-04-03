import "../shared/index.tsp";
import "./ratecard.tsp";
import "./plan.tsp";

namespace ProductCatalog;

/**
 * The status of the add-on defined by the effectiveFrom and effectiveTo properties.
 * - `draft`: The add-on has not yet been published and can be edited.
 * - `active`: The add-on is published and available for use.
 * - `archived`: The add-on is no longer available for use.
 */
@friendlyName("AddonStatus")
enum AddonStatus {
  Draft: "draft",
  Active: "active",
  Archived: "archived",
}

/**
 * The instanceType of the add-on.
 * - `single`: Can be added to a subscription only once.
 * - `multiple`: Can be added to a subscription more than once.
 */
@friendlyName("AddonInstanceType")
enum AddonInstanceType {
  Single: "single",
  Multiple: "multiple",
}

/**
 * Add-on allows extending subscriptions with compatible plans with additional ratecards.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "effective_from and effective_to are standard AIP temporal fields"
@friendlyName("Addon")
model Addon {
  ...Shared.Resource;

  /**
   * A key is a semi-unique string that is used to identify the add-on.
   * It is used to reference the latest `active` version of the add-on and is unique with the version number.
   */
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Key")
  key: Shared.ResourceKey;

  /**
   * Version of the add-on. Incremented when the add-on is updated.
   */
  @summary("Version")
  @minValue(1)
  @visibility(Lifecycle.Read)
  version: integer = 1;

  /**
   * The InstanceType of the add-ons. Can be "single" or "multiple".
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("The InstanceType of the add-ons. Can be \"single\" or \"multiple\".")
  instance_type: AddonInstanceType;

  /**
   * The currency code of the add-on.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Currency")
  currency: Currencies.CurrencyCode;

  /**
   * The date and time when the add-on becomes effective. When not specified, the add-on is a draft.
   */
  @visibility(Lifecycle.Read)
  @summary("Effective start date")
  effective_from?: Shared.DateTime;

  /**
   * The date and time when the add-on is no longer effective. When not specified, the add-on is effective indefinitely.
   */
  @visibility(Lifecycle.Read)
  @summary("Effective end date")
  effective_to?: Shared.DateTime;

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
  rate_cards: RateCard[];

  /**
   * List of validation errors.
   */
  @visibility(Lifecycle.Read)
  @summary("Validation errors")
  validation_errors?: ProductCatalogValidationError[];
}
