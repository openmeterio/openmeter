import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";
import "..";
import "../..";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;

@route("/api/v2/entitlements")
@tag("Entitlements")
@friendlyName("EntitlementsV2")
interface EntitlementsV2Endpoints {
  /**
   * List all entitlements for all the customers and features. This endpoint is intended for administrative purposes only.
   * To fetch the entitlements of a specific subject please use the /api/v2/customers/{customerIdOrKey}/entitlements endpoint.
   */
  @get
  @operationId("listEntitlementsV2")
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
     * Filtering by multiple customers.
     *
     * Usage: `?customerKeys=customer-1&customerKeys=customer-3`
     */
    @query(#{ explode: true })
    customerKeys?: string[],

    /**
     * Filtering by multiple customers.
     *
     * Usage: `?customerIds=01K4WAQ0J99ZZ0MD75HXR112H8&customerIds=01K4WAQ0J99ZZ0MD75HXR112H9`
     */
    @query(#{ explode: true })
    customerIds?: string[],

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
  ): PaginatedResponse<EntitlementV2> | CommonErrors;

  /**
   * Get entitlement by ID.
   */
  @get
  @operationId("getEntitlementByIdV2")
  @summary("Get entitlement by ID")
  get(
    @path
    entitlementId: ULID,
  ): EntitlementV2 | NotFoundError | CommonErrors;
}

/**
 * Entitlement templates are used to define the entitlements of a plan.
 * Features are omitted from the entitlement template, as they are defined in the rate card.
 */
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
@friendlyName("EntitlementV2")
union EntitlementV2 {
  metered: EntitlementMeteredV2,
  static: EntitlementStaticV2,
  boolean: EntitlementBooleanV2,
}

/**
 * Create inputs for entitlement
 */
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
@friendlyName("EntitlementV2CreateInputs")
union EntitlementV2CreateInputs {
  metered: EntitlementMeteredV2CreateInputs,
  static: EntitlementStaticCreateInputs,
  boolean: EntitlementBooleanCreateInputs,
}

/**
 * Customer fields for entitlements
 */
@friendlyName("EntitlementCustomerFields")
model EntitlementCustomerFields {
  /**
   * The identifier key unique to the customer
   */
  @example("customer-1")
  customerKey?: string;

  /**
   * The identifier unique to the customer
   */
  @example("01ARZ3NDEKTSV4RRFFQ69G5FAV")
  customerId: ULID;
}

/**
 * Issue after reset
 */
@friendlyName("IssueAfterReset")
model IssueAfterReset {
  /**
   * The initial grant amount
   */
  @summary("Initial grant amount")
  @minValue(0)
  amount: float64;

  /**
   * The priority of the issue after reset
   */
  @summary("Issue grant after reset priority")
  @minValue(1)
  @maxValue(255)
  priority?: uint8 = 1;
}

/**
 * Create inputs for metered entitlement
 */
@friendlyName("EntitlementMeteredV2CreateInputs")
model EntitlementMeteredV2CreateInputs {
  ...OmitProperties<
    EntitlementMeteredCreateInputs,
    "isUnlimited" | "issueAfterReset" | "issueAfterResetPriority"
  >;

  /**
   * You can grant usage automatically alongside the entitlement, the example scenario would be creating a starting balance.
   * If an amount is specified here, a grant will be created alongside the entitlement with the specified amount.
   * That grant will have it's rollover settings configured in a way that after each reset operation, the balance will return the original amount specified here.
   * Manually creating such a grant would mean having the "amount", "minRolloverAmount", and "maxRolloverAmount" fields all be the same.
   */
  #deprecated "Use issue.Amount instead, will be removed in next major version."
  @minValue(0)
  @summary("Initial grant amount")
  issueAfterReset?: float64;

  /**
   * Defines the grant priority for the default grant.
   */
  #deprecated "Use issue.Priority instead, will be removed in next major version."
  @minValue(1)
  @maxValue(255)
  @summary("Issue grant after reset priority")
  issueAfterResetPriority?: uint8 = 1;

  /**
   * Issue after reset
   */
  @summary("Issue after reset")
  issue?: IssueAfterReset;

  /**
   * Grants
   */
  @summary("Grants")
  grants?: GrantCreateInputV2[];
}

/**
 * Metered entitlements are useful for many different use cases, from setting up usage based access to implementing complex credit systems.
 * Access is determined based on feature usage using a balance calculation (the "usage allowance" provided by the issued grants is "burnt down" by the usage).
 */
@friendlyName("EntitlementMeteredV2")
model EntitlementMeteredV2 {
  type: EntitlementType.metered;
  ...OmitProperties<
    EntitlementMeteredV2CreateInputs,

      | "type"
      | "measureUsageFrom"
      | "metadata"
      | "usagePeriod"
      | "featureKey"
      | "featureId"
      | "currentUsagePeriod"
      | "grants"
  >;
  ...OmitProperties<
    EntitlementSharedFields,
    "type" | "currentUsagePeriod" | "usagePeriod" | "subjectKey"
  >;
  ...EntitlementMeteredCalculatedFields;
  ...EntitlementCustomerFields;
}

/**
 * Entitlement template of a boolean entitlement.
 */
@friendlyName("EntitlementBooleanV2")
model EntitlementBooleanV2 {
  type: EntitlementType.boolean;
  ...OmitProperties<
    EntitlementBooleanCreateInputs,
    "type" | "metadata" | "usagePeriod" | "featureKey" | "featureId"
  >;
  ...OmitProperties<EntitlementSharedFields, "type" | "subjectKey">;
  ...EntitlementCustomerFields;
}

/**
 * A static entitlement.
 */
@friendlyName("EntitlementStaticV2")
model EntitlementStaticV2 {
  type: EntitlementType.static;
  ...OmitProperties<
    EntitlementStaticCreateInputs,
    "type" | "metadata" | "usagePeriod" | "featureKey" | "featureId"
  >;
  ...OmitProperties<EntitlementSharedFields, "type" | "subjectKey">;
  ...EntitlementCustomerFields;
}
