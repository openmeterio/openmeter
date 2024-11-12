import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;
using TypeSpec.Versioning;

namespace OpenMeter.Entitlements;

/**
 * Order by options for grants.
 */
@friendlyName("GrantOrderBy")
enum GrantOrderBy {
  ID: "id",
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  CreatedAt: "createdAt",
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  UpdatedAt: "updatedAt",
}

@route("/api/v1/grants")
@tag("Entitlements (Beta)")
interface Grants {
  /**
   * List all grants for all the subjects and entitlements. This endpoint is intended for administrative purposes only.
   * To fetch the grants of a specific entitlement please use the /api/v1/subjects/{subjectKeyOrID}/entitlements/{entitlementOrFeatureID}/grants endpoint.
   * If page is provided that takes precedence and the paginated response is returned.
   */
  @get
  @operationId("listGrants")
  list(
    /**
     * Filtering by multiple features.
     *
     * Usage: `?feature=feature-1&feature=feature-2`
     */
    @query(#{ explode: true })
    feature?: string[],

    /**
     * Filtering by multiple subjects.
     *
     * Usage: `?subject=customer-1&subject=customer-2`
     */
    @query(#{ explode: true })
    subject?: string[],

    /**
     * Include deleted
     */
    @query includeDeleted?: boolean = false,

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryLimitOffset,
    ...OpenMeter.QueryOrdering<GrantOrderBy>,
  ): Grant[] | OpenMeter.PaginatedResponse<Grant> | OpenMeter.CommonErrors;

  /**
   * Voiding a grant means it is no longer valid, it doesn't take part in further balance calculations. Voiding a grant does not retroactively take effect, meaning any usage that has already been attributed to the grant will remain, but future usage cannot be burnt down from the grant.
   * For example, if you have a single grant for your metered entitlement with an initial amount of 100, and so far 60 usage has been metered, the grant (and the entitlement itself) would have a balance of 40. If you then void that grant, balance becomes 0, but the 60 previous usage will not be affected.
   */
  @delete
  @operationId("voidGrant")
  delete(@path grantId: string):
    | {
        @statusCode _: 204;
      }
    | OpenMeter.NotFoundError
    | OpenMeter.CommonErrors
    | OpenMeter.ConflictError;
}

/**
 * The grant.
 */
@friendlyName("EntitlementGrant")
model Grant {
  ...ResourceTimestamps;
  ...OmitProperties<GrantCreateInput, "recurrence">;

  /**
   * Readonly unique ULID identifier.
   */
  @example("01ARZ3NDEKTSV4RRFFQ69G5FAV")
  @visibility("read")
  id: ULID;

  /**
   * The unique entitlement ULID that the grant is associated with.
   */
  @example("01ARZ3NDEKTSV4RRFFQ69G5FAV")
  @visibility("read")
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
  @visibility("read")
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
@friendlyName("EntitlementGrantCreateInput")
model GrantCreateInput {
  /**
   * The amount to grant. Should be a positive number.
   */
  @minValue(0)
  @example(100.0)
  amount: float64;

  /**
   * The priority of the grant. Grants with higher priority are applied first.
   * Priority is a positive decimal numbers. With lower numbers indicating higher importance.
   * For example, a priority of 1 is more urgent than a priority of 2.
   * When there are several grants available for the same subject, the system selects the grant with the highest priority.
   * In cases where grants share the same priority level, the grant closest to its expiration will be used first.
   * In the case of two grants have identical priorities and expiration dates, the system will use the grant that was created first.
   */
  @minValue(1)
  @maxValue(255)
  @example(1)
  priority?: uint8;

  /**
   * Effective date for grants and anchor for recurring grants. Provided value will be ceiled to metering windowSize (minute).
   */
  effectiveAt: DateTime;

  /**
   * The grant expiration definition
   */
  expiration: ExpirationPeriod;

  /**
   * Grants are rolled over at reset, after which they can have a different balance compared to what they had before the reset.
   * Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))
   */
  @example(100.0)
  maxRolloverAmount?: float64 = 0;

  /**
   * Grants are rolled over at reset, after which they can have a different balance compared to what they had before the reset.
   * Balance after the reset is calculated as: Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))
   */
  @example(100.0)
  minRolloverAmount?: float64 = 0;

  /**
   * The grant metadata.
   */
  @example(#{ stripePaymentId: "pi_4OrAkhLvyihio9p51h9iiFnB" })
  metadata?: Metadata;

  /**
   * The subject of the grant.
   */
  recurrence?: RecurringPeriodCreateInput;
}

/**
 * The expiration duration enum
 */
@friendlyName("ExpirationDuration")
enum ExpirationDuration {
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  HOUR: "HOUR",
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  DAY: "DAY",
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  WEEK: "WEEK",
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  MONTH: "MONTH",
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  YEAR: "YEAR",
}

/**
 * The grant expiration definition
 */
@friendlyName("ExpirationPeriod")
model ExpirationPeriod {
  /**
   * The unit of time for the expiration period.
   */
  duration: ExpirationDuration;

  /**
   * The number of time units in the expiration period.
   */
  @example(12)
  count: integer;
}
