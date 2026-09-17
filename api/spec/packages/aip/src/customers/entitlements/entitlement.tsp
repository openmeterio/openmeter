import "@typespec/openapi";
import "@typespec/openapi3";
import "../../shared/index.tsp";
import "../../common/properties.tsp";
import "../../entitlements/index.tsp";
import "../../features/index.tsp";
import "../customer.tsp";

using TypeSpec.OpenAPI;

namespace Customers;

/**
 * Preset start of usage measurement for a metered entitlement.
 *
 * - `current_period_start`: usage is measured from the start of the current usage
 *   period.
 * - `now`: usage is measured from the entitlement creation time.
 */
@friendlyName("BillingEntitlementMeasureUsageFromPreset")
enum EntitlementMeasureUsageFromPreset {
  CurrentPeriodStart: "current_period_start",
  Now: "now",
}

/**
 * Defines the time from which usage is measured: either a preset or an explicit
 * timestamp.
 */
@oneOf
@friendlyName("BillingEntitlementMeasureUsageFrom")
@example(EntitlementMeasureUsageFromPreset.CurrentPeriodStart)
union EntitlementMeasureUsageFrom {
  Preset: EntitlementMeasureUsageFromPreset,
  Time: Shared.DateTime,
}

/**
 * Recurring period input with an interval and an optional anchor.
 */
@friendlyName("BillingEntitlementRecurringPeriodInput")
model EntitlementRecurringPeriodInput {
  /**
   * The interval duration in ISO 8601 format.
   */
  @summary("Interval")
  @example("P1M")
  interval: Shared.ISO8601Duration;

  /**
   * A date-time anchor to base the recurring period on. Defaults to the creation
   * time of the owning resource.
   */
  @summary("Anchor time")
  @example(Shared.DateTime.fromISO("2023-01-01T01:01:01.001Z"))
  anchor?: Shared.DateTime;
}

/**
 * Usage granted automatically alongside a metered entitlement. The grant is
 * configured so that after each reset the balance returns to `amount`; the typical
 * use case is a recurring starting balance.
 */
@friendlyName("BillingEntitlementIssueAfterReset")
model EntitlementIssueAfterReset {
  /**
   * The amount granted after each reset, in the feature's unit.
   */
  @summary("Initial grant amount")
  @example("100")
  amount: Shared.Numeric;

  /**
   * The priority of the grant created after each reset. Lower values have higher
   * priority.
   */
  @summary("Issue grant after reset priority")
  @minValue(1)
  @maxValue(255)
  priority?: uint8 = 1;
}

/**
 * A grant created together with a metered entitlement.
 */
@friendlyName("BillingEntitlementGrantCreateRequest")
model EntitlementGrantCreate {
  /**
   * The amount to grant, in the feature's unit. Must be positive.
   */
  @summary("Amount")
  @example("100")
  amount: Shared.Numeric;

  /**
   * The priority of the grant. Lower values have higher priority: a priority of 1 is
   * more urgent than a priority of 2. When several grants are available, the one
   * with the highest priority is consumed first; among equal priorities the one
   * closest to expiration wins, then the earliest created.
   */
  @summary("Priority")
  @minValue(1)
  @maxValue(255)
  @example(1)
  priority?: uint8;

  /**
   * Effective date for the grant and anchor for recurring grants. The value is
   * ceiled to the metering window size (minute).
   */
  @summary("Effective at")
  @example(Shared.DateTime.fromISO("2023-01-01T01:01:01.001Z"))
  effective_at: Shared.DateTime;

  /**
   * The duration after which the grant expires, counted from `effective_at`. Only
   * single-unit durations are accepted (for example `PT12H`, `P7D`, `P2W`, `P3M`,
   * `P1Y`). If omitted, the grant never expires.
   */
  @summary("Expires after")
  @example("P1M")
  expires_after?: Shared.ISO8601Duration;

  /**
   * Grants are rolled over at reset, after which they can have a different balance
   * compared to what they had before the reset. Balance after the reset is
   * calculated as
   * `MIN(max_rollover_amount, MAX(balance_before_reset, min_rollover_amount))`.
   * Defaults to `amount`.
   */
  @summary("Max rollover amount")
  @example("100")
  max_rollover_amount?: Shared.Numeric;

  /**
   * Grants are rolled over at reset, after which they can have a different balance
   * compared to what they had before the reset. Balance after the reset is
   * calculated as
   * `MIN(max_rollover_amount, MAX(balance_before_reset, min_rollover_amount))`.
   * Defaults to `0`.
   */
  @summary("Min rollover amount")
  @example("0")
  min_rollover_amount?: Shared.Numeric;

  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  @summary("Labels")
  labels?: Common.Labels;

  /**
   * The recurrence of the grant. When set, the grant amount is re-issued every
   * interval. The anchor defaults to `effective_at`.
   */
  @summary("Recurrence")
  recurrence?: EntitlementRecurringPeriodInput;
}

/**
 * Fields shared by every entitlement type.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "active_from and active_to are standard AIP temporal fields"
@friendlyName("BillingEntitlementBase")
model EntitlementBase<T extends global.Entitlements.EntitlementType> {
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * The type of the entitlement.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Type")
  type: T;

  /**
   * The feature the customer is entitled to use.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Feature")
  feature: Shared.ResourceReference<Features.Feature>;

  /**
   * The customer the entitlement belongs to.
   */
  @visibility(Lifecycle.Read)
  @summary("Customer")
  customer: Shared.ResourceReference<Customer>;

  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Labels")
  labels?: Common.Labels;

  /**
   * The usage period of the entitlement.
   */
  @visibility(Lifecycle.Read)
  @summary("Usage period")
  usage_period?: Shared.RecurringPeriod;

  /**
   * The current usage period of the entitlement.
   */
  @visibility(Lifecycle.Read)
  @summary("Current usage period")
  current_usage_period?: Shared.ClosedPeriod;

  /**
   * The time from which the entitlement is active.
   */
  @visibility(Lifecycle.Read)
  @summary("Active from")
  active_from: Shared.DateTime;

  /**
   * The time until which the entitlement is active. If not set, the entitlement is
   * active until deleted.
   */
  @visibility(Lifecycle.Read)
  @summary("Active to")
  active_to?: Shared.DateTime;

  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  @visibility(Lifecycle.Read)
  created_at: Shared.DateTime;

  /**
   * An ISO-8601 timestamp representation of entity last update date.
   */
  @visibility(Lifecycle.Read)
  updated_at: Shared.DateTime;

  /**
   * An ISO-8601 timestamp representation of entity deletion date.
   */
  @visibility(Lifecycle.Read)
  deleted_at?: Shared.DateTime;
}

/**
 * Metered entitlements are useful for many different use cases, from setting up
 * usage based access to implementing complex credit systems. Access is determined
 * based on feature usage using a balance calculation: the usage allowance provided
 * by the issued grants is burnt down by the usage.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "active_from and active_to are standard AIP temporal fields; issue_after_reset fields are deprecated aliases of issue"
@friendlyName("BillingEntitlementMetered")
model EntitlementMetered {
  ...OmitProperties<
    EntitlementBase<global.Entitlements.EntitlementType.Metered>,
    "usage_period" | "current_usage_period"
  >;

  /**
   * The usage period of the entitlement. The balance resets at the start of every
   * period.
   */
  @visibility(Lifecycle.Read)
  @summary("Usage period")
  usage_period: Shared.RecurringPeriod;

  /**
   * The current usage period of the entitlement.
   */
  @visibility(Lifecycle.Read)
  @summary("Current usage period")
  current_usage_period: Shared.ClosedPeriod;

  /**
   * If true, the customer can use the feature even if the entitlement is exhausted;
   * access remains granted.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Soft limit")
  is_soft_limit?: boolean = false;

  /**
   * Usage granted automatically after each reset. Cannot be combined with `grants`.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Issue after reset")
  issue?: EntitlementIssueAfterReset;

  /**
   * The amount granted automatically after each reset.
   *
   * @deprecated Use `issue.amount` instead.
   */
  #deprecated "Use issue.amount instead"
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Initial grant amount")
  @example("100")
  issue_after_reset?: Shared.Numeric;

  /**
   * The priority of the grant created after each reset.
   *
   * @deprecated Use `issue.priority` instead.
   */
  #deprecated "Use issue.priority instead"
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Issue grant after reset priority")
  @minValue(1)
  @maxValue(255)
  issue_after_reset_priority?: uint8 = 1;

  /**
   * If true, the overage is preserved at reset. If false, the usage is reset to 0.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Preserve overage at reset")
  preserve_overage_at_reset?: boolean = false;

  /**
   * The time from which usage is measured.
   */
  @visibility(Lifecycle.Read)
  @summary("Measure usage from")
  measure_usage_from: Shared.DateTime;

  /**
   * The time the last reset happened.
   */
  @visibility(Lifecycle.Read)
  @summary("Last reset")
  last_reset: Shared.DateTime;
}

/**
 * Static entitlements let you pass along a configuration while granting access,
 * for example "using this feature with X Y settings".
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "active_from and active_to are standard AIP temporal fields"
@friendlyName("BillingEntitlementStatic")
model EntitlementStatic {
  ...EntitlementBase<global.Entitlements.EntitlementType.Static>;

  /**
   * The entitlement config as a JSON value. Returned when checking entitlement
   * access; useful for configuring fine-grained access settings implemented in your
   * own system.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Config")
  @extension("x-go-type", "json.RawMessage")
  @example(#{ integrations: #["github"] })
  config: unknown;
}

/**
 * Boolean entitlements define static feature access, for example "can use SSO
 * authentication".
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "active_from and active_to are standard AIP temporal fields"
@friendlyName("BillingEntitlementBoolean")
model EntitlementBoolean {
  ...EntitlementBase<global.Entitlements.EntitlementType.Boolean>;
}

/**
 * An entitlement grants a customer access to a feature.
 */
@friendlyName("BillingEntitlement")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union Entitlement {
  @summary("Metered entitlement")
  metered: EntitlementMetered,

  @summary("Static entitlement")
  static: EntitlementStatic,

  @summary("Boolean entitlement")
  boolean: EntitlementBoolean,
}

/**
 * Metered entitlement create request.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "issue_after_reset fields are deprecated aliases of issue"
@withVisibility(Lifecycle.Create)
@friendlyName("CreateEntitlementMeteredRequest")
model CreateEntitlementMetered {
  ...OmitProperties<EntitlementMetered, "usage_period" | "measure_usage_from">;

  /**
   * The usage period of the entitlement. The balance resets at the start of every
   * period. The anchor defaults to the entitlement creation time.
   */
  @visibility(Lifecycle.Create)
  @summary("Usage period")
  usage_period: EntitlementRecurringPeriodInput;

  /**
   * Defines the time from which usage is measured. Defaults to the entitlement
   * creation time.
   */
  @visibility(Lifecycle.Create)
  @summary("Measure usage from")
  measure_usage_from?: EntitlementMeasureUsageFrom;

  /**
   * Grants created together with the entitlement. Cannot be combined with `issue`.
   */
  @visibility(Lifecycle.Create)
  @summary("Grants")
  grants?: EntitlementGrantCreate[];
}

/**
 * Static entitlement create request.
 */
@withVisibility(Lifecycle.Create)
@friendlyName("CreateEntitlementStaticRequest")
model CreateEntitlementStatic {
  ...OmitProperties<EntitlementStatic, "usage_period">;

  /**
   * The usage period of the entitlement. The anchor defaults to the entitlement
   * creation time.
   */
  @visibility(Lifecycle.Create)
  @summary("Usage period")
  usage_period?: EntitlementRecurringPeriodInput;
}

/**
 * Boolean entitlement create request.
 */
@withVisibility(Lifecycle.Create)
@friendlyName("CreateEntitlementBooleanRequest")
model CreateEntitlementBoolean {
  ...OmitProperties<EntitlementBoolean, "usage_period">;

  /**
   * The usage period of the entitlement. The anchor defaults to the entitlement
   * creation time.
   */
  @visibility(Lifecycle.Create)
  @summary("Usage period")
  usage_period?: EntitlementRecurringPeriodInput;
}

/**
 * Entitlement create request.
 */
@friendlyName("CreateEntitlementRequest")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union CreateEntitlement {
  @summary("Metered entitlement")
  metered: CreateEntitlementMetered,

  @summary("Static entitlement")
  static: CreateEntitlementStatic,

  @summary("Boolean entitlement")
  boolean: CreateEntitlementBoolean,
}
