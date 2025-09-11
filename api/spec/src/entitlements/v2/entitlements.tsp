import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";
import "..";
import "../..";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.Entitlements.V2;

@route("/api/v2/entitlements")
@tag("Entitlements")
interface Entitlements {
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

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryLimitOffset,
    ...OpenMeter.QueryOrdering<EntitlementOrderBy>,
  ): OpenMeter.PaginatedResponse<EntitlementV2> | OpenMeter.CommonErrors;

  /**
   * Get entitlement by id.
   */
  @get
  @operationId("getEntitlementByIdV2")
  @summary("Get entitlement by id")
  get(
    @path
    entitlementId: ULID,
  ): EntitlementV2 | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
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
 * Metered entitlements are useful for many different use cases, from setting up usage based access to implementing complex credit systems.
 * Access is determined based on feature usage using a balance calculation (the "usage allowance" provided by the issued grants is "burnt down" by the usage).
 */
@friendlyName("EntitlementMeteredV2")
model EntitlementMeteredV2 {
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
