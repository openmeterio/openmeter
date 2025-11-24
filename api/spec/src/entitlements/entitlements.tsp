import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;

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
@tag("Entitlements")
@friendlyName("Entitlements")
interface EntitlementsEndpoints {
  /**
   * List all entitlements for all the subjects and features. This endpoint is intended for administrative purposes only.
   * To fetch the entitlements of a specific subject please use the /api/v1/subjects/{subjectKeyOrID}/entitlements endpoint.
   * If page is provided that takes precedence and the paginated response is returned.
   *
   * ⚠️ __Deprecated__: Use [`GET /api/v2/entitlements`](#tag/entitlements/get/api/v2/entitlements) instead.
   */
  #deprecated "Use `GET /api/v2/entitlements` instead"
  #suppress "deprecated" "V1 Entitlements APIs will be removed on December 1st, 2025"
  @get
  @operationId("listEntitlements")
  @summary("List all entitlements")
  @extension("x-internal", true)
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

    /**
     * Exclude inactive entitlements in the response (those scheduled for later or earlier)
     */
    @query
    excludeInactive?: boolean = false,

    ...QueryPagination,
    ...QueryLimitOffset,
    ...QueryOrdering<EntitlementOrderBy>,
  ): {
    @statusCode _: 200;
    @body body: ListEntitlementsResult;
  } | CommonErrors;

  /**
   * Get entitlement by ID.
   *
   * ⚠️ __Deprecated__: Use [`GET /api/v2/entitlements/{entitlementId}`](#tag/entitlements/get/api/v2/entitlements/{entitlementId}) instead.
   */
  #deprecated "Use `GET /api/v2/entitlements/{entitlementId}` instead"
  #suppress "deprecated" "V1 Entitlements APIs will be removed on December 1st, 2025"
  @get
  @operationId("getEntitlementById")
  @summary("Get entitlement by ID")
  get(
    @path
    entitlementId: ULID,
  ): Entitlement | NotFoundError | CommonErrors;
}

// TODO: Remove in v2

/**
 * List entitlements result
 */
@oneOf
@friendlyName("ListEntitlementsResult")
union ListEntitlementsResult {
  Entitlement[],
  PaginatedResponse<Entitlement>,
}

/**
 * Type of the entitlement.
 */
#deprecated "Use EntitlementsV2 instead"
#suppress "deprecated" "V1 Entitlements APIs will be removed on December 1st, 2025"
@friendlyName("EntitlementType")
@extension("x-go-type", "string")
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
#deprecated "Use EntitlementV2 instead"
#suppress "deprecated" "V1 Entitlements APIs will be removed on December 1st, 2025"
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
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
  usagePeriod?: RecurringPeriodCreateInput;
}

/**
 * Create inpurs for metered entitlement
 */
#deprecated "Use EntitlementMeteredV2CreateInputs instead"
#suppress "deprecated" "V1 Entitlements APIs will be removed on December 1st, 2025"
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
  usagePeriod: RecurringPeriodCreateInput;

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
#deprecated "Use EntitlementStaticV2CreateInputs instead"
#suppress "deprecated" "V1 Entitlements APIs will be removed on December 1st, 2025"
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
#deprecated "Use EntitlementBooleanV2CreateInputs instead"
#suppress "deprecated" "V1 Entitlements APIs will be removed on December 1st, 2025"
@friendlyName("EntitlementBooleanCreateInputs")
model EntitlementBooleanCreateInputs {
  ...EntitlementCreateSharedFields;
  type: EntitlementType.boolean;
}

/**
 * Create inputs for entitlement
 */
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
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
  ...OmitProperties<
    EntitlementCreateSharedFields,
    "featureKey" | "featureId" | "usagePeriod"
  >;
  ...CadencedResource;

  /**
   * The annotations of the entitlement.
   */
  @example(#{ `subscription.id`: "sub_123" })
  @visibility(Lifecycle.Read)
  annotations?: Annotations;

  /**
   * Readonly unique ULID identifier.
   */
  @example("01ARZ3NDEKTSV4RRFFQ69G5FAV")
  @visibility(Lifecycle.Read)
  id: ULID;

  /**
   * The type of the entitlement.
   */
  @summary("Type")
  type: EntitlementType;

  /**
   * The identifier key unique to the subject.
   * NOTE: Subjects are being deprecated, please use the new customer APIs.
   */
  @example("customer-1")
  subjectKey: Key;

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
  currentUsagePeriod?: Period;

  /**
   * The defined usage period of the entitlement
   */
  usagePeriod?: RecurringPeriod;
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
  @visibility(Lifecycle.Read)
  lastReset: DateTime;

  /**
   * The current usage period.
   */
  @visibility(Lifecycle.Read)
  currentUsagePeriod: Period;

  /**
   * The time from which usage is measured. If not specified on creation, defaults to entitlement creation time.
   */
  @visibility(Lifecycle.Read)
  measureUsageFrom: DateTime;

  /**
   * THe usage period of the entitlement.
   */
  @visibility(Lifecycle.Read)
  usagePeriod: RecurringPeriod;
}

/**
 * Metered entitlements are useful for many different use cases, from setting up usage based access to implementing complex credit systems.
 * Access is determined based on feature usage using a balance calculation (the "usage allowance" provided by the issued grants is "burnt down" by the usage).
 */
#deprecated "Use EntitlementMeteredV2 instead"
#suppress "deprecated" "V1 Entitlements APIs will be removed on December 1st, 2025"
@friendlyName("EntitlementMetered")
model EntitlementMetered {
  type: EntitlementType.metered;
  ...OmitProperties<
    EntitlementMeteredCreateInputs,

      | "type"
      | "measureUsageFrom"
      | "metadata"
      | "usagePeriod"
      | "featureKey"
      | "featureId"
      | "currentUsagePeriod"
  >;
  ...OmitProperties<
    EntitlementSharedFields,
    "type" | "currentUsagePeriod" | "usagePeriod"
  >;
  ...EntitlementMeteredCalculatedFields;
}

/**
 * Entitlement template of a boolean entitlement.
 */
#deprecated "Use EntitlementBooleanV2 instead"
#suppress "deprecated" "V1 Entitlements APIs will be removed on December 1st, 2025"
@friendlyName("EntitlementBoolean")
model EntitlementBoolean {
  type: EntitlementType.boolean;
  ...OmitProperties<
    EntitlementBooleanCreateInputs,
    "type" | "metadata" | "usagePeriod" | "featureKey" | "featureId"
  >;
  ...OmitProperties<EntitlementSharedFields, "type">;
}

/**
 * A static entitlement.
 */
#deprecated "Use EntitlementStaticV2 instead"
#suppress "deprecated" "V1 Entitlements APIs will be removed on December 1st, 2025"
@friendlyName("EntitlementStatic")
model EntitlementStatic {
  type: EntitlementType.static;
  ...OmitProperties<
    EntitlementStaticCreateInputs,
    "type" | "metadata" | "usagePeriod" | "featureKey" | "featureId"
  >;
  ...OmitProperties<EntitlementSharedFields, "type">;
}

/**
 * Start of measurement options
 */
@friendlyName("MeasureUsageFromPreset")
@extension("x-enum-varnames", #["CurrentPeriodStart", "Now"])
enum MeasureUsageFromPreset {
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
  preset: MeasureUsageFromPreset,
  time: MeasureUsageFromTime,
}
