import "../shared/index.tsp";
import "../currencies/currency.tsp";
import "./ratecard.tsp";

namespace ProductCatalog;

/**
 * The status of a plan.
 *
 * - `draft`: The plan has not yet been published and can be edited.
 * - `active`: The plan is published and can be used in subscriptions.
 * - `archived`: The plan is no longer available for use.
 * - `scheduled`: The plan is scheduled to be published at a future date.
 */
@friendlyName("BillingPlanStatus")
enum PlanStatus {
  Draft: "draft",
  Active: "active",
  Archived: "archived",
  Scheduled: "scheduled",
}

/**
 * Plans provide a template for subscriptions.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "effective_from and effective_to are standard AIP temporal fields"
@friendlyName("BillingPlan")
model Plan {
  ...Shared.Resource;

  /**
   * A key is a semi-unique string that is used to identify the plan. It is used to
   * reference the latest `active` version of the plan and is unique with the version
   * number.
   */
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Key")
  key: Shared.ResourceKey;

  /**
   * Plans are versioned to allow you to make changes without affecting running
   * subscriptions.
   */
  @visibility(Lifecycle.Read)
  @summary("Version")
  @minValue(1)
  version: integer = 1;

  /**
   * The currency code of the plan.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Currency")
  currency: Currencies.CurrencyCode;

  /**
   * The billing cadence for subscriptions using this plan.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Billing cadence")
  billing_cadence: Shared.ISO8601Duration;

  /**
   * Whether pro-rating is enabled for this plan.
   */
  // TODO: WIP — pro-rating mode and exact semantics will be defined in a later iteration.
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Pro-rating enabled")
  pro_rating_enabled?: boolean = true;

  /**
   * The date and time when the plan becomes `active`. When not specified, the plan
   * is in `draft` status.
   */
  @visibility(Lifecycle.Read)
  @summary("Effective start date")
  effective_from?: Shared.DateTime;

  /**
   * A scheduled date and time when the plan becomes `archived`. When not specified,
   * the plan is in `active` status indefinitely.
   */
  @visibility(Lifecycle.Read)
  @summary("Effective end date")
  effective_to?: Shared.DateTime;

  /**
   * The status of the plan. Computed based on the effective start and end dates:
   *
   * - `draft`: `effective_from` is not set.
   * - `scheduled`: `now < effective_from`.
   * - `active`: `effective_from <= now` and (`effective_to` is not set or
   *   `now < effective_to`).
   * - `archived`: `effective_to <= now`.
   */
  @visibility(Lifecycle.Read)
  @summary("Status")
  status: PlanStatus;

  /**
   * The plan phases define the pricing ramp for a subscription. A phase switch
   * occurs only at the end of a billing period. At least one phase is required.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Plan phases")
  @minItems(1)
  phases: PlanPhase[];

  /**
   * Settlement mode for plan.
   *
   * Values:
   *
   * - `credit_then_invoice`: Credits are applied first, then any remainder is
   *   invoiced.
   * - `credit_only`: Usage is settled exclusively against credits.
   */
  @visibility(Lifecycle.Read)
  @summary("Settlement Mode")
  settlement_mode: ProductCatalog.SettlementMode;

  /**
   * List of validation errors in `draft` state that prevent the plan from being
   * published.
   */
  @visibility(Lifecycle.Read)
  @summary("Validation errors")
  validation_errors?: ProductCatalogValidationError[];
}

/**
 * Plan create request. `settlement_mode` is re-declared as optional with a
 * `credit_then_invoice` default, applied only on create.
 */
@friendlyName("CreatePlanRequest")
model CreatePlanRequestBody {
  ...Shared.CreateRequest<OmitProperties<Plan, "settlement_mode">>;

  /**
   * Settlement mode for the plan. Defaults to `credit_then_invoice` when omitted.
   */
  @visibility(Lifecycle.Create)
  @summary("Settlement Mode")
  settlement_mode?: ProductCatalog.SettlementMode = ProductCatalog.SettlementMode.CreditThenInvoice;
}

/**
 * Plan upsert (update) request. `settlement_mode` is re-declared as optional with
 * no default, so an omitted value leaves the plan's existing settlement mode
 * unchanged.
 */
@friendlyName("UpsertPlanRequest")
model UpsertPlanRequestBody {
  ...Shared.UpsertRequest<OmitProperties<Plan, "settlement_mode">>;

  /**
   * Settlement mode for the plan. When omitted, the existing settlement mode is
   * preserved.
   */
  @visibility(Lifecycle.Create, Lifecycle.Update)
  @summary("Settlement Mode")
  settlement_mode?: ProductCatalog.SettlementMode;
}

/**
 * Validation errors providing detailed description of the issue.
 */
@friendlyName("ProductCatalogValidationError")
model ProductCatalogValidationError is Shared.BaseError<string> {
  /**
   * The path to the field.
   */
  @visibility(Lifecycle.Read)
  @summary("Field")
  @example("addons/pro/ratecards/token/featureKey")
  field: string;
}

/**
 * The plan phase or pricing ramp allows changing a plan's rate cards over time as
 * a subscription progresses.
 */
@friendlyName("BillingPlanPhase")
model PlanPhase {
  ...PickProperties<
    Shared.ResourceWithKey,
    "key" | "name" | "description" | "labels"
  >;

  /**
   * The duration of the phase. When not specified, the phase runs indefinitely. Only
   * the last phase may omit the duration.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Duration")
  duration?: Shared.ISO8601Duration;

  /**
   * The rate cards of the plan.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Rate cards")
  rate_cards: RateCard[];
}
