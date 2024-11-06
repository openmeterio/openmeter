import "@typespec/http";
import "../types.tsp";
import "./ratecards.tsp";

namespace OpenMeter.ProductCatalog;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

/**
 * The status of a plan.
 */
@friendlyName("PlanStatus")
enum PlanStatus {
  draft: "draft",
  active: "active",
  archived: "archived",
  scheduled: "scheduled",
}

/**
 * References an exact plan.
 */
@friendlyName("PlanReference")
model PlanReference {
  /**
   * The plan key.
   */
  key: Key;

  /**
   * The plan version.
   */
  version: integer;
}

/**
 * Plans provide a template for subscriptions.
 */
@friendlyName("Plan")
model Plan {
  ...UniqueResource;

  /**
   * Version of the plan. Incremented when the plan is updated.
   */
  @summary("Version")
  @minValue(1)
  @visibility("read")
  version: integer = 1;

  /**
   * The currency code of the plan.
   */
  @summary("Currency")
  currency: CurrencyCode = "USD";

  /**
   * The date and time when the plan becomes effective. When not specified, the plan is a draft.
   */
  @summary("Effective start date")
  effectiveFrom?: DateTime;

  /**
   * The date and time when the plan is no longer effective. When not specified, the plan is effective indefinitely.
   */
  @summary("Effective end date")
  effectiveTo?: DateTime;

  /**
   * The status of the plan.
   * Computed based on the effective start and end dates:
   *  - draft = no effectiveFrom
   *  - active = effectiveFrom <= now < effectiveTo
   *  - archived / inactive = effectiveTo <= now
   *  - scheduled = now < effectiveFrom < effectiveTo
   */
  @summary("Status")
  @visibility("read")
  status: PlanStatus;

  /**
   * The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription progresses.
   * A phase switch occurs only at the end of a billing period, ensuring that a single subscription invoice will not include charges from different phase prices.
   */
  @summary("Plan phases")
  @minItems(1)
  phases: PlanPhase[];
}

/**
 * The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription progresses.
 */
@friendlyName("PlanPhase")
model PlanPhase {
  ...OmitProperties<Resource, "id">;

  /**
   * Unique key among the plan phases. Used to reference the phase in the subscription & plan.
   */
  key: Key;

  /**
   * The rate cards of the plan.
   */
  @summary("Rate cards")
  rateCards: RateCard[];

  /**
   * The time after which the plan starts compared to subscription start
   */
  @summary("Start after")
  @encode(DurationKnownEncoding.ISO8601)
  @example(duration.fromISO("P1Y1D"))
  startAfter: duration | null;

  /**
   * The discounts on the plan.
   */
  @summary("Discounts")
  discounts?: AppliedDiscount[];

  // /**
  //  * Predefined overrides of the plan that can apply changes to the rate cards, discounts, and other properties based on the selected variant.
  //  */
  // @summary("Variants")
  // variants?: PlanVariantOverridePreset[] = #[];
}

/**
 * A plan variant.
 * A set of properties that can be used to select a specific configuration of a plan.
 */
@friendlyName("PlanVariant")
model PlanVariant {
  currency: CurrencyCode;

  // additional properties are not yet supported
  // ...Record<string>
}

/**
 * Plan variant override preset.
 */
@friendlyName("PlanVariantOverridePreset")
model PlanVariantOverridePreset {
  /**
   * The variant of the plan.
   */
  @example(#{ currency: "USD" })
  @summary("Variant")
  variant: PlanVariant;

  /**
   * The currency code.
   */
  @summary("Currency")
  currency?: CurrencyCode;

  /**
   * The rate card overrides.
   */
  @summary("Rate card overrides")
  rateCards?: {
    /**
     * Add rate cards to the plan.
     */
    @summary("Add rate card")
    add?: RateCard[];

    /**
     * Remove rate cards from the plan.
     */
    @summary("Remove rate card")
    remove?: Key[];

    // /**
    //  * Update rate cards in the plan.
    //  */
    // @summary("Update rate card")
    // update?: Record<RateCardOverride>;
  };

  /*
   * The discount on the plan.
   */
  discount?: AppliedDiscount;
}

/**
 * Order by options for plans.
 */
@friendlyName("PlanOrderBy")
enum PlanOrderBy {
  id: "id",
  key: "key",
  version: "version",
  createdAt: "create_at",
  updatedAt: "updated_at",
}

/**
 * Order by options for plan phases.
 */
@friendlyName("PhasesOrderBy")
enum PhasesOrderBy {
  key: "key",
  startAfter: "start_after",
}
