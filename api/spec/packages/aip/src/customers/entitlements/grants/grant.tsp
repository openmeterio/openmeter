import "@typespec/openapi";
import "@typespec/openapi3";
import "../../../shared/index.tsp";
import "../../../common/properties.tsp";

using TypeSpec.OpenAPI;

namespace Customers;

/**
 * A grant issued for a metered entitlement. Each grant adds its amount to the
 * entitlement's balance from its effective time until it expires, and usage is
 * deducted from the grants in priority order.
 *
 * Grants are immutable, so the balance is deterministic regardless of when it is
 * queried. Deleting a grant ends it at the time of the deletion.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "expires_after is the configured duration and expires_at is the calculated timestamp"
@friendlyName("BillingEntitlementGrant")
model EntitlementGrant {
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * The ID of the entitlement the grant belongs to.
   */
  @visibility(Lifecycle.Read)
  @summary("Entitlement ID")
  entitlement_id: Shared.ULID;

  /**
   * The granted amount, in the feature's unit.
   */
  @visibility(Lifecycle.Read)
  @summary("Amount")
  @example("100")
  amount: Shared.Numeric;

  /**
   * The priority of the grant. Lower values are consumed first: a grant with
   * priority 1 is consumed before one with priority 2. Among equal priorities, the
   * grant closest to expiration is consumed first, then the earliest created.
   */
  @visibility(Lifecycle.Read)
  @summary("Priority")
  @example(1)
  priority: uint8;

  /**
   * The time the grant takes effect. It is also the anchor of the recurrence for
   * recurring grants.
   */
  @visibility(Lifecycle.Read)
  @summary("Effective at")
  @example(Shared.DateTime.fromISO("2023-01-01T01:01:01.001Z"))
  effective_at: Shared.DateTime;

  /**
   * The duration after which the grant expires, counted from `effective_at`. Always
   * a single-unit duration (for example `PT12H`, `P7D`, `P2W`, `P3M`, `P1Y`). Absent
   * when the grant never expires.
   */
  @visibility(Lifecycle.Read)
  @summary("Expires after")
  @example("P1M")
  expires_after?: Shared.ISO8601Duration;

  /**
   * The time the grant expires, calculated from `effective_at` and `expires_after`.
   * The grant is no longer in effect from this time. Absent when the grant never
   * expires.
   */
  @visibility(Lifecycle.Read)
  @summary("Expires at")
  @example(Shared.DateTime.fromISO("2023-02-01T01:01:01.001Z"))
  expires_at?: Shared.DateTime;

  /**
   * The maximum balance the grant carries over at reset. The balance after a reset
   * is `MIN(max_rollover_amount, MAX(balance_before_reset, min_rollover_amount))`.
   */
  @visibility(Lifecycle.Read)
  @summary("Max rollover amount")
  @example("100")
  max_rollover_amount: Shared.Numeric;

  /**
   * The minimum balance the grant carries over at reset. The balance after a reset
   * is `MIN(max_rollover_amount, MAX(balance_before_reset, min_rollover_amount))`.
   */
  @visibility(Lifecycle.Read)
  @summary("Min rollover amount")
  @example("0")
  min_rollover_amount: Shared.Numeric;

  /**
   * The recurrence of the grant. When set, the grant amount is re-issued every
   * interval, anchored at `effective_at`. Absent for non-recurring grants.
   */
  @visibility(Lifecycle.Read)
  @summary("Recurrence")
  recurrence?: Shared.RecurringPeriod;

  /**
   * The next time the grant recurs. Absent for non-recurring grants.
   */
  @visibility(Lifecycle.Read)
  @summary("Next recurrence")
  @example(Shared.DateTime.fromISO("2023-02-01T01:01:01.001Z"))
  next_recurrence?: Shared.DateTime;

  /**
   * The time the grant was voided. A voided grant is no longer in effect from this
   * time.
   */
  @visibility(Lifecycle.Read)
  @summary("Voided at")
  @example(Shared.DateTime.fromISO("2023-01-15T01:01:01.001Z"))
  voided_at?: Shared.DateTime;

  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  @visibility(Lifecycle.Read)
  @summary("Labels")
  labels?: Common.Labels;

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
