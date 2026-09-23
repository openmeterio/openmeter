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
 * Preset for the time from which usage is measured.
 *
 * - `current_period_start`: the start of the current usage period.
 * - `now`: the entitlement creation time.
 */
@friendlyName("BillingEntitlementMeasureUsageFromPreset")
enum EntitlementMeasureUsageFromPreset {
  CurrentPeriodStart: "current_period_start",
  Now: "now",
}

/**
 * The time from which usage is measured, as a preset or an explicit timestamp.
 */
@oneOf
@friendlyName("BillingEntitlementMeasureUsageFrom")
@example(EntitlementMeasureUsageFromPreset.CurrentPeriodStart)
union EntitlementMeasureUsageFrom {
  Preset: EntitlementMeasureUsageFromPreset,
  Time: Shared.DateTime,
}

/**
 * Usage granted automatically after each reset of a metered entitlement. The
 * balance returns to `amount` after every reset.
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
 * A grant to issue for a metered entitlement.
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
   * The priority of the grant. Lower values have higher priority. Grants are
   * consumed in priority order, then by closest expiration, then by earliest
   * creation.
   */
  @summary("Priority")
  @minValue(1)
  @maxValue(255)
  @example(1)
  priority?: uint8;

  /**
   * The time the grant becomes effective and the anchor for recurring grants. The
   * value is rounded up to the next minute.
   */
  @summary("Effective at")
  @example(Shared.DateTime.fromISO("2023-01-01T01:01:01.001Z"))
  effective_at: Shared.DateTime;

  /**
   * The duration after which the grant expires, counted from `effective_at`. Only
   * single-unit durations are accepted, such as `PT12H`, `P7D`, `P2W`, `P3M`, or
   * `P1Y`. If omitted, the grant never expires.
   */
  @summary("Expires after")
  @example("P1M")
  expires_after?: Shared.ISO8601Duration;

  /**
   * The maximum balance carried over at reset. The balance after a reset is
   * `MIN(max_rollover_amount, MAX(balance_before_reset, min_rollover_amount))`.
   * Defaults to `amount`.
   */
  @summary("Max rollover amount")
  @example("100")
  max_rollover_amount?: Shared.Numeric;

  /**
   * The minimum balance carried over at reset. The balance after a reset is
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
   * The recurrence of the grant. When set, the amount is issued again every
   * interval. The anchor defaults to `effective_at`.
   */
  @summary("Recurrence")
  recurrence?: Shared.RecurringPeriodInput;
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
   * The time the entitlement was created.
   */
  @visibility(Lifecycle.Read)
  created_at: Shared.DateTime;

  /**
   * The time the entitlement was last updated.
   */
  @visibility(Lifecycle.Read)
  updated_at: Shared.DateTime;

  /**
   * The time the entitlement was deleted.
   */
  @visibility(Lifecycle.Read)
  deleted_at?: Shared.DateTime;
}

/**
 * A metered entitlement grants a usage allowance for a feature. Access is
 * determined by the balance: the allowance provided by grants is burnt down by
 * usage.
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
   * If true, the customer keeps access to the feature after the balance is
   * exhausted.
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
   * The time of the last reset.
   */
  @visibility(Lifecycle.Read)
  @summary("Last reset")
  last_reset: Shared.DateTime;
}

/**
 * A static entitlement grants access to a feature together with a configuration.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "active_from and active_to are standard AIP temporal fields"
@friendlyName("BillingEntitlementStatic")
model EntitlementStatic {
  ...EntitlementBase<global.Entitlements.EntitlementType.Static>;

  /**
   * The entitlement configuration as a JSON value. Returned when checking
   * entitlement access.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Config")
  @extension("x-go-type", "json.RawMessage")
  @example(#{ integrations: #["github"] })
  config: unknown;
}

/**
 * A boolean entitlement grants access to a feature.
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
  usage_period: Shared.RecurringPeriodInput;

  /**
   * The time from which usage is measured. Defaults to the entitlement creation
   * time.
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
  usage_period?: Shared.RecurringPeriodInput;
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
  usage_period?: Shared.RecurringPeriodInput;
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

namespace Entitlements {
  /**
   * Request body for resetting the usage of a metered entitlement.
   */
  @friendlyName("ResetCustomerEntitlementUsageRequest")
  model ResetCustomerEntitlementUsageRequest {
    /**
     * The time the reset takes effect. Defaults to the current time and cannot be in
     * the future. Truncated to the minute.
     */
    effective_at?: Shared.DateTime;

    /**
     * Whether the usage period anchor is kept. When false, the anchor moves to
     * `effective_at`.
     */
    retain_anchor?: boolean = false;

    /**
     * Whether overage carries over into the new usage period. Defaults to the
     * entitlement's own setting.
     */
    preserve_overage?: boolean;
  }
}
