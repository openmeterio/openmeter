import "@typespec/openapi";
import "@typespec/openapi3";
import "../../../shared/index.tsp";
import "../../../common/properties.tsp";

using TypeSpec.OpenAPI;

namespace Customers;

/**
 * A grant issued for a metered entitlement. Grants define the usage allowance the
 * entitlement's balance is burnt down from: each grant is in effect between its
 * effective time and its expiration, and grants are consumed in priority order.
 *
 * Grants are immutable once created, so the balance is deterministic regardless of
 * when it is queried. They can only be deleted, which ends them at the time of the
 * deletion.
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
   * The priority of the grant. Lower values have higher priority: a priority of 1 is
   * more urgent than a priority of 2. When several grants are available, the one
   * with the highest priority is consumed first; among equal priorities the one
   * closest to expiration wins, then the earliest created.
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
   * The grant is no longer in effect at this time. Absent when the grant never
   * expires.
   */
  @visibility(Lifecycle.Read)
  @summary("Expires at")
  @example(Shared.DateTime.fromISO("2023-02-01T01:01:01.001Z"))
  expires_at?: Shared.DateTime;

  /**
   * Grants are rolled over at reset, after which they can have a different balance
   * compared to what they had before the reset. Balance after the reset is
   * calculated as
   * `MIN(max_rollover_amount, MAX(balance_before_reset, min_rollover_amount))`.
   */
  @visibility(Lifecycle.Read)
  @summary("Max rollover amount")
  @example("100")
  max_rollover_amount: Shared.Numeric;

  /**
   * Grants are rolled over at reset, after which they can have a different balance
   * compared to what they had before the reset. Balance after the reset is
   * calculated as
   * `MIN(max_rollover_amount, MAX(balance_before_reset, min_rollover_amount))`.
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
