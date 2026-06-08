import "../shared/index.tsp";
import "../entitlements/entitlements.tsp";

namespace ProductCatalog;

/**
 * Entitlement template configured on a rate card. The feature is taken from the
 * rate card itself, so it is omitted here.
 */
@friendlyName("BillingRateCardEntitlement")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union RateCardEntitlement {
  @summary("Metered entitlement")
  metered: RateCardMeteredEntitlement,

  @summary("Static entitlement")
  static: RateCardStaticEntitlement,

  @summary("Boolean entitlement")
  boolean: RateCardBooleanEntitlement,
}

/**
 * The initial grant issued on entitlement creation and re-issued on each reset.
 */
@friendlyName("BillingRateCardIssueAfterReset")
model IssueAfterReset {
  /**
   * The initial grant amount granted alongside the entitlement.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Initial grant amount")
  @minValue(0)
  amount: float64;

  /**
   * The priority of the default grant.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Grant priority")
  @minValue(1)
  @maxValue(255)
  priority?: uint8 = 1;
}

/**
 * The entitlement template of a metered entitlement.
 */
@friendlyName("BillingRateCardMeteredEntitlement")
model RateCardMeteredEntitlement {
  /**
   * The type of the entitlement template.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Type")
  type: Entitlements.EntitlementType.Metered;

  /**
   * If soft limit is true, the subject can use the feature even if the entitlement
   * is exhausted; access remains granted.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Soft limit")
  is_soft_limit?: boolean = false;

  /**
   * The structured initial grant for the entitlement.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Issue after reset")
  issue?: IssueAfterReset;

  /**
   * If true, the overage is preserved across resets; otherwise usage resets to 0.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Preserve overage at reset")
  preserve_overage_at_reset?: boolean = false;

  /**
   * The reset interval of the metered entitlement in ISO8601 format. Defaults to the
   * billing cadence of the rate card.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Usage period")
  usage_period?: Shared.ISO8601Duration;
}

/**
 * The entitlement template of a static entitlement.
 */
@friendlyName("BillingRateCardStaticEntitlement")
model RateCardStaticEntitlement {
  /**
   * The type of the entitlement template.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Type")
  type: Entitlements.EntitlementType.Static;

  /**
   * A JSON-parsable config returned when checking entitlement access. Useful for
   * configuring fine-grained access settings implemented in your own system.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Config")
  @encode("json")
  config: string;
}

/**
 * The entitlement template of a boolean entitlement.
 */
@friendlyName("BillingRateCardBooleanEntitlement")
model RateCardBooleanEntitlement {
  /**
   * The type of the entitlement template.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Type")
  type: Entitlements.EntitlementType.Boolean;
}
