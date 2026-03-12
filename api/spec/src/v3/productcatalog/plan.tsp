import "../shared/index.tsp";
import "./ratecard.tsp";

namespace ProductCatalog;

/**
 * The status of a plan.
 */
@friendlyName("BillingPlanStatus")
enum PlanStatus {
  /**
   * The plan has not yet been published and can be edited.
   */
  Draft: "draft",

  /**
   * The plan is published and can be used in subscriptions.
   */
  Active: "active",

  /**
   * The plan is no longer available for use.
   */
  Archived: "archived",

  /**
   * The plan is scheduled to be published at a future date.
   */
  Scheduled: "scheduled",
}

/**
 * Plans provide a template for subscriptions.
 */
@friendlyName("BillingPlan")
model Plan {
  ...Shared.ResourceWithKey;

  /**
   * Plans are versioned to allow you to make changes without affecting running subscriptions.
   */
  @summary("Version")
  @minValue(1)
  @visibility(Lifecycle.Read)
  version: integer = 1;

  /**
   * The currency code of the plan.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Currency")
  currency: Shared.CurrencyCode;

  /**
   * The billing cadence for subscriptions using this plan.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Billing cadence")
  billing_cadence: Shared.ISO8601Duration;

  /**
   * The date and time when the plan becomes `active`.
   * When not specified, the plan is in `draft` status.
   */
  @visibility(Lifecycle.Read)
  @summary("Effective start date")
  effective_from?: Shared.DateTime;

  /**
   * A scheduled date and time when the plan becomes `archived`.
   * When not specified, the plan is in `active` status indefinitely.
   */
  @visibility(Lifecycle.Read)
  @summary("Effective end date")
  effective_to?: Shared.DateTime;

  /**
   * The status of the plan.
   * Computed based on the effective start and end dates:
   *  - draft = no effectiveFrom
   *  - active = effectiveFrom <= now < effectiveTo
   *  - archived / inactive = effectiveTo <= now
   *  - scheduled = now < effectiveFrom < effectiveTo
   */
  @summary("Status")
  @visibility(Lifecycle.Read)
  status: PlanStatus;

  /**
   * List of validation errors in `draft` state that prevent the plan from being published.
   */
  @visibility(Lifecycle.Read)
  @summary("Validation errors")
  validation_errors?: PlanValidationError[];
}

/**
 * Validation errors providing detailed description of the issue.
 */
@friendlyName("BillingPlanValidationError")
model PlanValidationError {
  /**
   * The path to the field in the plan definition.
   */
  @visibility(Lifecycle.Read)
  @example("addons/pro/ratecards/token/featureKey")
  field: string;

  /**
   * The machine readable code of the error.
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

/**
 * The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription progresses.
 */
@friendlyName("BillingPlanPhase")
model PlanPhase {
  ...PickProperties<
    Shared.ResourceWithKey,
    "key" | "name" | "description" | "labels"
  >;

  /**
   * The duration of the phase.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Duration")
  duration: Shared.ISO8601Duration;

  /**
   * The rate cards of the plan.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Rate cards")
  rateCards: RateCard[];
}
