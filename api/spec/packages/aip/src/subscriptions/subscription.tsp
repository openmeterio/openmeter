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
    "customer_id" | "plan_id" | "billing_anchor"
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
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "cost basis mode and pins are separate subscription properties"
@friendlyName("BillingSubscription")
model Subscription {
  ...OmitProperties<Shared.Resource, "name" | "description">;

  /**
   * The customer ID of the subscription.
   */
  @visibility(Lifecycle.Read)
  @summary("Customer ID")
  customer_id: Shared.ULID;

  /**
   * The plan ID of the subscription. Set if subscription is created from a plan.
   */
  @visibility(Lifecycle.Read)
  @summary("Plan ID")
  plan_id?: Shared.ULID;

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
}
