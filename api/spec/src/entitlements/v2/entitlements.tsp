import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";
import "..";
import "../..";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.Entitlements.V2;

/**
 * Entitlement templates are used to define the entitlements of a plan.
 * Features are omitted from the entitlement template, as they are defined in the rate card.
 */
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
@friendlyName("EntitlementV2")
union EntitlementV2 {
  metered: EntitlementMetered,
  static: EntitlementStatic,
  boolean: EntitlementBoolean,
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
