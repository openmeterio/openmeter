import "@typespec/openapi3";
import "../../shared/index.tsp";
import "../../meters/query.tsp";

namespace Customers.Entitlements;

using TypeSpec.OpenAPI;

/**
 * The meter query granularities the usage history can be grouped into. Sub-hour
 * windows are too expensive to compute and monthly windows are not supported.
 */
@friendlyName("BillingEntitlementHistoryWindowSize")
@oneOf
union EntitlementHistoryWindowSize {
  /**
   * 1 hour windows.
   */
  @summary("1 hour")
  Hour: Meters.MeterQueryGranularityHour,

  /**
   * 1 day windows.
   */
  @summary("1 day")
  Day: Meters.MeterQueryGranularityDay,
}

/**
 * Balance and usage history of a metered entitlement.
 */
@friendlyName("BillingEntitlementHistory")
model EntitlementHistory {
  /**
   * Usage in half-open windows of the requested size, aligned to the requested time
   * zone. Empty windows are included; windows before usage measurement began are
   * omitted.
   */
  @visibility(Lifecycle.Read)
  windowed_history: EntitlementHistoryWindow[];

  /**
   * Periods in which grants were consumed in a fixed order. A new segment starts
   * whenever grant priorities change or a usage period starts.
   */
  @visibility(Lifecycle.Read)
  burndown_history: EntitlementBurndownSegment[];
}

/**
 * Usage and balance of a single history window.
 */
@friendlyName("BillingEntitlementHistoryWindow")
model EntitlementHistoryWindow {
  /**
   * The period the window covers.
   */
  @visibility(Lifecycle.Read)
  period: Shared.ClosedPeriod;

  /**
   * The usage recorded in the window.
   */
  @visibility(Lifecycle.Read)
  @example("100")
  usage: Shared.Numeric;

  /**
   * The entitlement balance at the start of the window.
   */
  @visibility(Lifecycle.Read)
  @example("100")
  balance_at_start: Shared.Numeric;
}

/**
 * A period in which grants were consumed in a fixed order.
 */
@friendlyName("BillingEntitlementBurndownSegment")
model EntitlementBurndownSegment {
  /**
   * The period the segment covers.
   */
  @visibility(Lifecycle.Read)
  period: Shared.ClosedPeriod;

  /**
   * The usage recorded in the segment.
   */
  @visibility(Lifecycle.Read)
  @example("100")
  usage: Shared.Numeric;

  /**
   * The usage in the segment not covered by any grant.
   */
  @visibility(Lifecycle.Read)
  @example("0")
  overage: Shared.Numeric;

  /**
   * The entitlement balance at the start and at the end of the segment.
   */
  @visibility(Lifecycle.Read)
  balance: EntitlementBurndownBalance;

  /**
   * The balance of each active grant at the start and at the end of the segment,
   * keyed by grant ID.
   */
  @visibility(Lifecycle.Read)
  grant_balances: EntitlementBurndownGrantBalances;

  /**
   * The grants consumed in the segment and the usage taken from each.
   */
  @visibility(Lifecycle.Read)
  grant_usages: EntitlementGrantUsage[];
}

/**
 * Entitlement balance at the boundaries of a burndown segment.
 */
@friendlyName("BillingEntitlementBurndownBalance")
model EntitlementBurndownBalance {
  /**
   * The balance at the start of the segment.
   */
  @visibility(Lifecycle.Read)
  @example("100")
  start: Shared.Numeric;

  /**
   * The balance at the end of the segment.
   */
  @visibility(Lifecycle.Read)
  @example("50")
  end: Shared.Numeric;
}

/**
 * Grant balances at the boundaries of a burndown segment, keyed by grant ID.
 */
@friendlyName("BillingEntitlementBurndownGrantBalances")
model EntitlementBurndownGrantBalances {
  /**
   * The balance of each active grant at the start of the segment.
   */
  @visibility(Lifecycle.Read)
  @example(#{ `01G65Z755AFWAKHE12NY0CQ9FH`: "100" })
  start: Record<Shared.Numeric>;

  /**
   * The balance of each active grant at the end of the segment.
   */
  @visibility(Lifecycle.Read)
  @example(#{ `01G65Z755AFWAKHE12NY0CQ9FH`: "50" })
  end: Record<Shared.Numeric>;
}

/**
 * Usage taken from a single grant.
 */
@friendlyName("BillingEntitlementGrantUsage")
model EntitlementGrantUsage {
  /**
   * The ID of the grant.
   */
  @visibility(Lifecycle.Read)
  @example("01G65Z755AFWAKHE12NY0CQ9FH")
  grant_id: Shared.ULID;

  /**
   * The usage taken from the grant.
   */
  @visibility(Lifecycle.Read)
  @example("50")
  usage: Shared.Numeric;
}
