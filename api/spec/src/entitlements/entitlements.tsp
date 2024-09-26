import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

// TODO: does this have to be in a separate namespace?
namespace OpenMeter.Entitlements;

/**
 * Order by options for entitlements.
 */
@friendlyName("EntitlementOrderBy")
enum EntitlementOrderBy {
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  CreatedAt: "createdAt",
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  UpdatedAt: "updatedAt",
}

@route("/api/v1/entitlements")
@tag("Entitlements (Experimental)")
interface Entitlements {
  /**
   * List all entitlements for all the subjects and features. This endpoint is intended for administrative purposes only.
   * To fetch the entitlements of a specific subject please use the /api/v1/subjects/{subjectKeyOrID}/entitlements endpoint.
   * If page is provided that takes precedence and the paginated response is returned.
   */
  @get
  @operationId("listEntitlements")
  @summary("List all entitlements")
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
     * Filtering by multiple entitlement types.
     *
     * Usage: `?entitlementType=metered&entitlementType=boolean`
     */
    @query(#{ explode: true })
    entitlementType?: EntitlementType[],

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryLimitOffset,
    ...OpenMeter.QueryOrdering<EntitlementOrderBy>,
  ): Entitlement[] | OpenMeter.PaginatedResponse<Entitlement> | OpenMeter.CommonErrors;

  /**
   * Get entitlement by id.
   */
  @get
  @operationId("getEntitlementById")
  get(
    @path
    entitlementId: ULID,
  ): Entitlement | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

/**
 * Type of the entitlement.
 */
@friendlyName("EntitlementType")
enum EntitlementType {
  metered: "metered",
  boolean: "boolean",
  static: "static",
}

// NOTE: use OmitProperties<T> for templates once entitlement models are defined

/**
 * Entitlement templates are used to define the entitlements of a plan.
 * Features are omitted from the entitlement template, as they are defined in the rate card.
 */
@discriminator("type")
@friendlyName("Entitlement")
union Entitlement {
  metered: EntitlementMetered,
  static: EntitlementStatic,
  boolean: EntitlementBoolean,
}

/**
 * Shared fields for entitlement creation
 */
@friendlyName("EntitlementCreateSharedFields")
model EntitlementCreateSharedFields {
  /**
   * The feature the subject is entitled to use.
   * Either featureKey or featureId is required.
   */
  @example("example-feature-key")
  featureKey?: Key;

  /**
   * The feature the subject is entitled to use.
   * Either featureKey or featureId is required.
   */
  @example("01ARZ3NDEKTSV4RRFFQ69G5FAV")
  featureId?: ULID;

  /**
   * Additional metadata for the feature.
   */
  metadata?: Metadata;

  /**
   * The usage period associated with the entitlement.
   */
  usagePeriod?: RecurringPeriod;
}

/**
 * Create inpurs for metered entitlement
 */
@friendlyName("EntitlementMeteredCreateInputs")
model EntitlementMeteredCreateInputs {
  ...OmitProperties<EntitlementCreateSharedFields, "usagePeriod">;
  type: EntitlementType.metered;

  /**
   * If softLimit=true the subject can use the feature even if the entitlement is exhausted, hasAccess will always be true.
   */
  @summary("Soft limit")
  isSoftLimit?: boolean = false;

  /**
   * Deprecated, ignored by the backend. Please use isSoftLimit instead; this field will be removed in the future.
   */
  #deprecated "Use isSoftLimit instead"
  isUnlimited?: boolean = false;

  /**
   * The usage period associated with the entitlement.
   */
  usagePeriod: RecurringPeriod;

  /**
   * Defines the time from which usage is measured. If not specified on creation, defaults to entitlement creation time.
   */
  measureUsageFrom?: MeasureUsageFrom;

  /**
   * You can grant usage automatically alongside the entitlement, the example scenario would be creating a starting balance.
   * If an amount is specified here, a grant will be created alongside the entitlement with the specified amount.
   * That grant will have it's rollover settings configured in a way that after each reset operation, the balance will return the original amount specified here.
   * Manually creating such a grant would mean having the "amount", "minRolloverAmount", and "maxRolloverAmount" fields all be the same.
   */
  @minValue(0)
  @summary("Initial grant amount")
  issueAfterReset?: float64;

  /**
   * Defines the grant priority for the default grant.
   */
  @minValue(1)
  @maxValue(255)
  @summary("Issue grant after reset priority")
  issueAfterResetPriority?: uint8 = 1;

  /**
   * If true, the overage is preserved at reset. If false, the usage is reset to 0.
   */
  @summary("Preserve overage at reset")
  preserveOverageAtReset?: boolean = false;
}

/**
 * Create inputs for static entitlement
 */
@friendlyName("EntitlementStaticCreateInputs")
model EntitlementStaticCreateInputs {
  ...EntitlementCreateSharedFields;
  type: EntitlementType.static;

  /**
   * The JSON parsable config of the entitlement. This value is also returned when checking entitlement access and it is useful for configuring fine-grained access settings to the feature, implemented in your own system. Has to be an object.
   */
  @example("{ \"integrations\": [\"github\"] }")
  @encode("json")
  config: string;
}

/**
 * Create inputs for boolean entitlement
 */
@friendlyName("EntitlementBooleanCreateInputs")
model EntitlementBooleanCreateInputs {
  ...EntitlementCreateSharedFields;
  type: EntitlementType.boolean;
}

/**
 * Create inputs for entitlement
 */
@discriminator("type")
@friendlyName("EntitlementCreateInputs")
union EntitlementCreateInputs {
  metered: EntitlementMeteredCreateInputs,
  static: EntitlementStaticCreateInputs,
  boolean: EntitlementBooleanCreateInputs,
}

/**
 * Shared fields of the entitlement templates.
 */
@friendlyName("EntitlementBaseTemplate")
model EntitlementSharedFields {
  ...ResourceTimestamps;
  ...OmitProperties<EntitlementCreateSharedFields, "featureKey" | "featureId">;

  /**
   * The type of the entitlement.
   */
  @summary("Type")
  type: EntitlementType;

  /**
   * The identifier key unique to the subject
   */
  @example("customer-1")
  subjectkey: Key;

  /**
   * The feature the subject is entitled to use.
   */
  @example("example-feature-key")
  featureKey: Key;

  /**
   * The feature the subject is entitled to use.
   */
  @example("01ARZ3NDEKTSV4RRFFQ69G5FAV")
  featureId: ULID;

  /**
   * The current usage period.
   */
  currentUsagePeriod: Period;
}

/**
 * Calculated fields for metered entitlements.
 */
@friendlyName("EntitlementMeteredCalculatedFields")
model EntitlementMeteredCalculatedFields {
  /**
   * The time the last reset happened.
   */
  @example(DateTime.fromISO("2023-01-01T01:01:01.001Z"))
  @visibility("read")
  lastReset: DateTime;

  /**
   * The current usage period.
   */
  @visibility("read")
  currentUsagePeriod: Period;

  /**
   * The time from which usage is measured. If not specified on creation, defaults to entitlement creation time.
   */
  @visibility("read")
  measureUsageFrom: DateTime;
}

/**
 * Metered entitlements are useful for many different use cases, from setting up usage based access to implementing complex credit systems.
 * Access is determined based on feature usage using a balance calculation (the "usage allowance" provided by the issued grants is "burnt down" by the usage).
 */
@friendlyName("EntitlementMetered")
model EntitlementMetered {
  type: EntitlementType.metered;
  ...OmitProperties<
    EntitlementMeteredCreateInputs,
    "type" | "measureUsageFrom" | "metadata" | "usagePeriod" | "featureKey" | "featureId" | "currentUsagePeriod"
  >;
  ...OmitProperties<EntitlementMeteredCalculatedFields, "currentUsagePeriod">;
  ...OmitProperties<EntitlementSharedFields, "type">;
}

/**
 * Entitlement template of a boolean entitlement.
 */
@friendlyName("EntitlementBoolean")
model EntitlementBoolean extends EntitlementSharedFields {
  type: EntitlementType.boolean;
  ...OmitProperties<EntitlementBooleanCreateInputs, "type" | "metadata" | "usagePeriod" | "featureKey" | "featureId">;
  ...OmitProperties<EntitlementSharedFields, "type">;
}

/**
 * A static entitlement.
 */
@friendlyName("EntitlementStatic")
model EntitlementStatic {
  type: EntitlementType.static;
  ...OmitProperties<EntitlementStaticCreateInputs, "type" | "metadata" | "usagePeriod" | "featureKey" | "featureId">;
  ...OmitProperties<EntitlementSharedFields, "type">;
}

/**
 * Start of measurement options
 */
@friendlyName("MeasureUsageFromEnum")
enum MeasureUsageFromEnum {
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  CURRENT_PERIOD_START: "CURRENT_PERIOD_START",
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  NOW: "NOW",
}

@friendlyName("MeasureUsageFromTime")
scalar MeasureUsageFromTime extends DateTime;

/**
 * Measure usage from
 */
@friendlyName("MeasureUsageFrom")
@oneOf
union MeasureUsageFrom {
  `enum`: MeasureUsageFromEnum,
  time: MeasureUsageFromTime,
}
