import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

// TODO: does this have to be in a separate namespace?
namespace OpenMeter.Entitlements;

@route("/api/v1/entitlements")
@tag("Entitlements (Experimental)")
interface Entitlements {}

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
@friendlyName("EntitlementTemplate")
union EntitlementTemplate {
  metered: EntitlementMeteredTemplate,
  static: EntitlementStaticTemplate,
  boolean: EntitlementBooleanTemplate,
}

/**
 * Shared fields of the entitlement templates.
 */
@friendlyName("EntitlementBaseTemplate")
model EntitlementBaseTemplate {
  /**
   * The type of the entitlement.
   */
  @summary("Type")
  type: EntitlementType;

  /**
   * Additional metadata for the entitlement.
   */
  @summary("Metadata")
  metadata?: Metadata;
}

/**
 * The entitlement template with a metered entitlement.
 */
@friendlyName("EntitlementMeteredTemplate")
model EntitlementMeteredTemplate extends EntitlementBaseTemplate {
  type: EntitlementType.metered;

  /**
   * If softLimit is `true` the subject can use the feature even if the entitlement is exhausted, hasAccess will always be true.
   */
  @summary("Soft limit")
  isSoftLimit?: boolean = false;

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

  /**
   * The interval of the metered entitlement.
   * Defaults to the billing cycle of the plan.
   */
  @summary("Interval")
  usagePeriod?: Interval;
}

/**
 * Entitlement template of a boolean entitlement.
 */
@friendlyName("EntitlementBooleanTemplate")
model EntitlementBooleanTemplate extends EntitlementBaseTemplate {
  type: EntitlementType.boolean;
}

/**
 * Entitlement template of a static entitlement.
 */
@friendlyName("EntitlementStaticTemplate")
model EntitlementStaticTemplate extends EntitlementBaseTemplate {
  type: EntitlementType.static;

  /**
   * The JSON parsable config of the entitlement.
   */
  @summary("Config")
  @example("{ \"integrations\": [\"github\"] }")
  @encode("json")
  config: string;
}
