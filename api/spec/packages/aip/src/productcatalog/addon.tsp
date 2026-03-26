import "../shared/index.tsp";
import "./ratecard.tsp";

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
@friendlyName("Addon")
model Addon {
  ...Shared.ResourceWithKey;

  /**
   * Set of key-value pairs managed by the system. Cannot be modified by user.
   */
  @visibility(Lifecycle.Read)
  @summary("Annotations")
  annotations?: Record<unknown>;

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
  attributes?: Record<unknown>;
}
