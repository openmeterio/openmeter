import "@typespec/openapi";
import "../shared/index.tsp";
import "../productcatalog/ratecard.tsp";

using TypeSpec.OpenAPI;

namespace Subscriptions;

/**
 * Subscription status.
 */
@friendlyName("BillingSubscriptionStatus")
enum SubscriptionStatus {
  Active: "active",
  Inactive: "inactive",
  Canceled: "canceled",
  Scheduled: "scheduled",
}

/**
 * Controls how custom-currency cost bases are selected for the subscription.
 */
@friendlyName("BillingSubscriptionCostBasisMode")
enum SubscriptionCostBasisMode {
  /** Resolve the cost basis effective at the charge's full service period start. */
  Dynamic: "dynamic",

  /** Pin each custom-currency pair when it is introduced to the subscription. */
  Pinned: "pinned",
}

/**
 * A cost basis pinned to a custom-currency pair for the subscription.
 */
@friendlyName("BillingSubscriptionCostBasisPin")
model SubscriptionCostBasisPin {
  /**
   * The managed custom currency ID.
   */
  @visibility(Lifecycle.Read)
  custom_currency_id: Shared.ULID;

  /**
   * The fiat currency in which the subscription is invoiced.
   */
  @visibility(Lifecycle.Read)
  invoice_currency: Shared.CurrencyCode;

  /**
   * The pinned cost basis resource ID.
   */
  @visibility(Lifecycle.Read)
  cost_basis_id: Shared.ULID;
}

/**
 * Subscription edit timing. When immediate, the requested changes take effect
 * immediately. When next_billing_cycle, the requested changes take effect at the
 * next billing cycle.
 */
@friendlyName("BillingSubscriptionEditTimingEnum")
enum SubscriptionEditTimingEnum {
  Immediate: "immediate",
  NextBillingCycle: "next_billing_cycle",
}

/**
 * Subscription edit timing defined when the changes should take effect. If the
 * provided configuration is not supported by the subscription, an error will be
 * returned.
 */
@oneOf
@friendlyName("BillingSubscriptionEditTiming")
@example(SubscriptionEditTimingEnum.Immediate)
union SubscriptionEditTiming {
  Enum: SubscriptionEditTimingEnum,
  Custom: Shared.DateTime,
}

/**
 * Subscription create request.
 */
@friendlyName("BillingSubscriptionCreate")
model SubscriptionCreate {
  ...Shared.CreateRequest<OmitProperties<
    Subscription,
    "customer_id" | "plan" | "billing_anchor"
  >>;

  /**
   * The customer to create the subscription for.
   */
  customer: {
    /**
     * The ID of the customer to create the subscription for.
     *
     * Either customer ID or customer key must be provided. If both are provided, the
     * ID will be used.
     */
    @visibility(Lifecycle.Create)
    @summary("Customer ID")
    id?: Shared.ULID;

    /**
     * The key of the customer to create the subscription for.
     *
     * Either customer ID or customer key must be provided. If both are provided, the
     * ID will be used.
     */
    @visibility(Lifecycle.Create)
    @summary("Customer Key")
    key?: Shared.ExternalResourceKey;
  };

  /**
   * The plan reference of the subscription.
   */
  plan: {
    /**
     * The plan ID of the subscription. Set if subscription is created from a plan.
     *
     * ID or Key of the plan is required if creating a subscription from a plan. If
     * both are provided, the ID will be used.
     */
    @visibility(Lifecycle.Create)
    @summary("Plan ID")
    id?: Shared.ULID;

    /**
     * The plan Key of the subscription, if any. Set if subscription is created from a
     * plan.
     *
     * ID or Key of the plan is required if creating a subscription from a plan. If
     * both are provided, the ID will be used.
     */
    @visibility(Lifecycle.Create)
    @summary("Plan Key")
    key?: Shared.ResourceKey;

    /**
     * The plan version of the subscription, if any. If not provided, the latest
     * version of the plan will be used.
     */
    @visibility(Lifecycle.Create)
    @summary("Plan Version")
    version?: integer;
  };

  /**
   * A billing anchor is the fixed point in time that determines the subscription's
   * recurring billing cycle. It affects when charges occur and how prorations are
   * calculated. Common anchors:
   *
   * - Calendar month (1st of each month): `2025-01-01T00:00:00Z`
   * - Subscription anniversary (day customer signed up)
   * - Custom date (customer-specified day)
   *
   * If not provided, the subscription will be created with the subscription's
   * creation time as the billing anchor.
   */
  @visibility(Lifecycle.Create)
  @summary("Billing anchor")
  billing_anchor?: Shared.DateTime;

  /**
   * Controls how custom-currency cost bases are selected for the subscription.
   */
  @visibility(Lifecycle.Create)
  @summary("Cost basis mode")
  cost_basis_mode?: SubscriptionCostBasisMode = SubscriptionCostBasisMode.Dynamic;
}

/**
 * Subscription.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "cost_basis, active, and billing fields are separate top-level subscription properties, not sub-objects"
@friendlyName("BillingSubscription")
model Subscription {
  ...OmitProperties<Shared.Resource, "name" | "description">;

  /**
   * Display name of the subscription. Defaults to the plan name when the
   * subscription is created from a plan.
   */
  @visibility(Lifecycle.Read)
  @minLength(1)
  @maxLength(256)
  @summary("Name")
  name: string;

  /**
   * Optional description of the subscription.
   */
  @visibility(Lifecycle.Read)
  @maxLength(1024)
  @summary("Description")
  description?: string;

  /**
   * An ISO-8601 timestamp representation of when the subscription became (or will
   * become) active.
   */
  @visibility(Lifecycle.Read)
  @summary("Active from")
  active_from: Shared.DateTime;

  /**
   * An ISO-8601 timestamp representation of when the subscription stops being
   * active. Open-ended when not set.
   */
  @visibility(Lifecycle.Read)
  @summary("Active to")
  active_to?: Shared.DateTime;

  /**
   * The customer ID of the subscription.
   */
  @visibility(Lifecycle.Read)
  @summary("Customer ID")
  customer_id: Shared.ULID;

  /**
   * The plan ID of the subscription. Set if subscription is created from a plan.
   *
   * @deprecated Use `plan` instead, which also includes the plan key and version.
   */
  #deprecated "Use plan instead"
  @visibility(Lifecycle.Read)
  @summary("Plan ID")
  plan_id?: Shared.ULID;

  /**
   * The plan the subscription was created from, if any. Includes the plan key and
   * version so clients can resolve the exact plan revision.
   */
  @visibility(Lifecycle.Read)
  @summary("Plan")
  plan?: SubscriptionPlanReference;

  /**
   * The fiat currency in which the subscription is invoiced.
   */
  @visibility(Lifecycle.Read)
  @summary("Invoice currency")
  invoice_currency: Shared.CurrencyCode;

  /**
   * Controls whether custom-currency cost bases are resolved dynamically or pinned
   * when their currency pair is introduced to the subscription.
   */
  @visibility(Lifecycle.Read)
  @summary("Cost basis mode")
  cost_basis_mode: SubscriptionCostBasisMode = SubscriptionCostBasisMode.Dynamic;

  /**
   * Cost bases pinned to custom-currency pairs for this subscription.
   */
  @visibility(Lifecycle.Read)
  @summary("Pinned cost bases")
  cost_basis_pins: SubscriptionCostBasisPin[];

  /**
   * The billing cadence of the subscription in ISO-8601 duration format. Defines how
   * often the customer is billed. Examples: `P1M` (monthly), `P3M` (quarterly),
   * `P1Y` (annually).
   */
  @visibility(Lifecycle.Read)
  @summary("Billing cadence")
  billing_cadence: Shared.ISO8601Duration;

  /**
   * The pro-rating configuration of the subscription.
   */
  @visibility(Lifecycle.Read)
  @summary("Pro-rating configuration")
  pro_rating_config?: SubscriptionProRatingConfig;

  /**
   * A billing anchor is the fixed point in time that determines the subscription's
   * recurring billing cycle. It affects when charges occur and how prorations are
   * calculated. Common anchors:
   *
   * - Calendar month (1st of each month): `2025-01-01T00:00:00Z`
   * - Subscription anniversary (day customer signed up)
   * - Custom date (customer-specified day)
   */
  @visibility(Lifecycle.Read)
  @summary("Billing anchor")
  billing_anchor: Shared.DateTime;

  /**
   * The status of the subscription.
   */
  @visibility(Lifecycle.Read)
  @summary("Status")
  status: SubscriptionStatus;

  /**
   * Settlement mode for billing.
   *
   * Values:
   *
   * - `credit_then_invoice`: Credits are applied first, then any remainder is
   *   invoiced.
   * - `credit_only`: Usage is settled exclusively against credits.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  @summary("Settlement Mode")
  settlement_mode?: ProductCatalog.SettlementMode;

  /**
   * The current aligned billing period. Present only when the subscription is active
   * and aligned.
   */
  @visibility(Lifecycle.Read)
  @summary("Current billing period")
  current_period?: Shared.ClosedPeriod;

  /**
   * The phases of the subscription in chronological order. A phase groups the rate
   * cards that are in effect for a segment of the subscription's lifetime.
   */
  @visibility(Lifecycle.Read)
  @summary("Phases")
  phases: SubscriptionPhase[];
}

/**
 * A reference to the plan a subscription was created from, pinned to an exact
 * revision.
 */
@friendlyName("BillingSubscriptionPlanReference")
model SubscriptionPlanReference {
  /**
   * The plan ID (exact revision).
   */
  @visibility(Lifecycle.Read)
  @summary("Plan ID")
  id: Shared.ULID;

  /**
   * The plan key. References the plan across versions.
   */
  @visibility(Lifecycle.Read)
  @summary("Plan key")
  key: Shared.ResourceKey;

  /**
   * The plan version.
   */
  @visibility(Lifecycle.Read)
  @summary("Plan version")
  @minValue(1)
  version: integer;
}

/**
 * The pro-rating configuration of a subscription.
 */
@friendlyName("BillingSubscriptionProRatingConfig")
model SubscriptionProRatingConfig {
  /**
   * Whether pro-rating is enabled.
   */
  @visibility(Lifecycle.Read)
  @summary("Enabled")
  enabled: boolean;

  /**
   * How pro-rating is calculated when enabled.
   */
  @visibility(Lifecycle.Read)
  @summary("Mode")
  mode: ProductCatalog.ProrationMode;
}

/**
 * A subscription phase groups the rate cards in effect for a segment of the
 * subscription's lifetime. Analogous to plan phases.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "active_from and active_to are standard AIP temporal fields"
@friendlyName("BillingSubscriptionPhase")
model SubscriptionPhase {
  ...Shared.ResourceWithKey;

  /**
   * An ISO-8601 timestamp representation of when the phase becomes active.
   */
  @visibility(Lifecycle.Read)
  @summary("Active from")
  active_from: Shared.DateTime;

  /**
   * An ISO-8601 timestamp representation of when the phase stops being active.
   * Open-ended for the last phase.
   */
  @visibility(Lifecycle.Read)
  @summary("Active to")
  active_to?: Shared.DateTime;

  /**
   * The rate cards in effect for this phase, resolved to the version active at the
   * queried time (the currently active version for the current phase, the first
   * version for future phases, and the last version for past phases).
   */
  @visibility(Lifecycle.Read)
  @summary("Items")
  items: SubscriptionItem[];
}

/**
 * A subscription item pins a rate card to a cadence within a subscription phase.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "active_from and active_to are standard AIP temporal fields"
@friendlyName("BillingSubscriptionItem")
model SubscriptionItem {
  /**
   * The unique identifier of the subscription item instance.
   */
  @visibility(Lifecycle.Read)
  @summary("ID")
  id: Shared.ULID;

  /**
   * An ISO-8601 timestamp representation of when this item version becomes active.
   */
  @visibility(Lifecycle.Read)
  @summary("Active from")
  active_from: Shared.DateTime;

  /**
   * An ISO-8601 timestamp representation of when this item version stops being
   * active.
   */
  @visibility(Lifecycle.Read)
  @summary("Active to")
  active_to?: Shared.DateTime;

  /**
   * The rate card describing what the customer gets and pays for this item.
   */
  @visibility(Lifecycle.Read)
  @summary("Rate card")
  rate_card: ProductCatalog.RateCard;
}
