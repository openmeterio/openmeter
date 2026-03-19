import "@typespec/http";
import "./ratecard.tsp";

namespace ProductCatalog;

/**
 * The status of the add-on defined by the effectiveFrom and effectiveTo properties.
 */
@friendlyName("AddonStatus")
enum AddonStatus {
  Draft: "draft",
  Active: "active",
  Archived: "archived",
}

/**
 * The instanceType of the add-on.
 * Single instance add-ons can be added to subscription only once while add-ons with multiple type can be added more then once.
 */
@friendlyName("AddonInstanceType")
enum AddonInstanceType {
  Single: "single",
  Multiple: "multiple",
}

/**
 * Add-on allows extending subscriptions with compatible plans with additional ratecards.
 */
@friendlyName("Addon")
model Addon {
  ...Shared.ResourceWithKey;

  /**
   * Set of key-value pairs managed by the system. Cannot be modified by user.
   */
  @visibility(Lifecycle.Read)
  @summary("Annotations")
  annotations?: Shared.Annotations;

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
  instance_type: AddonInstanceType;

  /**
   * The currency code of the add-on.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Currency")
  currency: Shared.CurrencyCode = "USD";

  /**
   * The date and time when the add-on becomes effective. When not specified, the add-on is a draft.
   */
  @visibility(Lifecycle.Read)
  @summary("Effective start date")
  from?: Shared.DateTime;

  /**
   * The date and time when the add-on is no longer effective. When not specified, the add-on is effective indefinitely.
   */
  @visibility(Lifecycle.Read)
  @summary("Effective end date")
  to?: Shared.DateTime;

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
  validation_errors?: ValidationError[];
}

// TODO(roland.spekker): figure out if we need the validation error type or not
/**
 * Validation errors providing detailed description of the issue.
 */
@friendlyName("ValidationError")
model ValidationError {
  /**
   * The path to the field.
   */
  @visibility(Lifecycle.Read)
  @example("addons/pro/ratecards/token/featureKey")
  field: string;

  /**
   * The machine readable description of the error.
   */
  @visibility(Lifecycle.Read)
  @example("invalid_feature_key")
  code: string;

  /**
   * The human readable description of the error.
   */
  @visibility(Lifecycle.Read)
  @example("not found feature by key")
  message: string;

  /**
   * Additional attributes.
   */
  @visibility(Lifecycle.Read)
  attributes?: Shared.Annotations;
}
