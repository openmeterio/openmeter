import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";
import "..";
import "../..";

namespace OpenMeter.Entitlements.V2;

/**
 * The grant.
 */
@friendlyName("EntitlementGrantV2")
model GrantV2 {
  ...ResourceTimestamps;
  ...OmitProperties<GrantCreateInputV2, "recurrence">;

  /**
   * Readonly unique ULID identifier.
   */
  @example("01ARZ3NDEKTSV4RRFFQ69G5FAV")
  @visibility(Lifecycle.Read)
  id: ULID;

  /**
   * The unique entitlement ULID that the grant is associated with.
   */
  @example("01ARZ3NDEKTSV4RRFFQ69G5FAV")
  @visibility(Lifecycle.Read)
  entitlementId: string;

  /**
   * The next time the grant will recurr.
   */
  @example(DateTime.fromISO("2023-01-01T01:01:01.001Z"))
  nextRecurrence?: DateTime;

  /**
   * The time the grant expires.
   */
  @example(DateTime.fromISO("2023-01-01T01:01:01.001Z"))
  @visibility(Lifecycle.Read)
  expiresAt?: DateTime;

  /**
   * The time the grant was voided.
   */
  @example(DateTime.fromISO("2023-01-01T01:01:01.001Z"))
  voidedAt?: DateTime;

  /**
   * The recurrence period of the grant.
   */
  recurrence?: RecurringPeriod;
}

/**
 * The grant creation input.
 */
@friendlyName("EntitlementGrantCreateInputV2")
model GrantCreateInputV2 {
  ...OmitProperties<
    OpenMeter.Entitlements.GrantCreateInput,
    "maxRolloverAmount" | "metadata" | "expiration"
  >;

  /**
   * Grants are rolled over at reset, after which they can have a different balance compared to what they had before the reset. The default value equals grant amount.
   * Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))
   */
  @example(100.0)
  maxRolloverAmount?: float64;

  /**
   * The grant expiration definition. If no expiration is provided, the grant can be active indefinitely.
   */
  expiration?: ExpirationPeriod;

  /**
   * The grant metadata.
   */
  #deprecated "Use annotations instead, will be removed in next major version."
  @example(#{ stripePaymentId: "pi_4OrAkhLvyihio9p51h9iiFnB" })
  metadata?: Metadata;

  /**
   * The grant metadata.
   */
  @example(#{ stripePaymentId: "pi_4OrAkhLvyihio9p51h9iiFnB" })
  annotations?: Annotations;
}
