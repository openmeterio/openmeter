import "@typespec/http";
import "./types.tsp";
import "./ratecards.tsp";

namespace OpenMeter;

using TypeSpec.Http;

@route("/api/v1/plans")
@tag("Plans")
interface Plans {}

/**
 * The status of a plan.
 */
@friendlyName("PlanStatus")
enum PlanStatus {
  draft: "draft",
  active: "active",
  archived: "archived",
}

/**
 * Plans provide a template for subscriptions.
 */
@friendlyName("Plan")
model Plan extends UniqueResource {
  /**
   * Version of the plan. Incremented when the plan is updated.
   */
  @summary("Version")
  @minValue(1)
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
  effectiveStartDate?: DateTime;

  /**
   * The date and time when the plan is no longer effective. When not specified, the plan is effective indefinitely.
   */
  @summary("Effective end date")
  effectiveEndDate?: DateTime;

  /**
   * The status of the plan.
   * Computed based on the effective start and end dates:
   *  - draft = no effectiveStartDate
   *  - active = effectiveStartDate <= now < effectiveEndDate
   *  - archived / inactive = effectiveEndDate <= now
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
  /**
   * Interval after the subscription starts to transition to the phase.
   * When null, the phase starts immediately after the subscription starts.
   */
  @summary("Start after")
  startAfter: Interval | null;

  /**
   * The rate cards of the plan.
   */
  @summary("Rate cards")
  rateCards: RateCard[];

  /**
   * The discount on the plan.
   */
  @summary("Discount")
  discount?: PlanDiscount;

  /**
   * Predefined overrides of the plan that can apply changes to the rate cards, discounts, and other properties based on the selected variant.
   */
  @summary("Variants")
  variants?: PlanVariantOverridePreset[] = #[];
}

/**
 * A discount on plan. One of: percentage or amount.
 */
@friendlyName("PlanDiscount")
@discriminator("type")
union PlanDiscount {
  @summary("Percentage discount")
  percentage: PlanDiscountPercentage,

  @summary("Amount discount")
  amount: PlanDiscountAmount,
}

/**
 * Percentage discount on plan.
 */
@friendlyName("PlanDiscountPercentage")
model PlanDiscountPercentage {
  ...DiscountPercentage;

  /**
   * The rate cards that the discount applies to.
   * When not specified, the discount applies to all rate cards.
   */
  @summary("Applies To")
  appliesToRateCards?: Key[];
}

/**
 * Amount discount on plan.
 */
@friendlyName("PlanDiscountAmount")
model PlanDiscountAmount {
  ...DiscountAmount;

  /**
   * The rate cards that the discount applies to.
   * When not specified, the discount applies to all rate cards.
   */
  @summary("Applies To")
  appliesToRateCards?: Key[];
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

    /**
     * Update rate cards in the plan.
     */
    @summary("Update rate card")
    update?: Record<RateCardOverride>;
  };

  /**
   * The discount on the plan.
   */
  @summary("Discount")
  discount?: PlanDiscount;
}
