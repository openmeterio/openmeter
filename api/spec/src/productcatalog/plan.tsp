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
 * References an exact plan defaulting to the current active version.
 */
@friendlyName("PlanReferenceInput")
model PlanReferenceInput {
  /**
   * The plan key.
   */
  key: Key;

  /**
   * The plan version.
   */
  version?: integer;
}

/**
 * References an exact plan.
 */
@friendlyName("PlanReference")
model PlanReference {
  /**
   * The plan ID.
   */
  id: ULID;

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
   * Alignment configuration for the plan.
   */
  alignment?: Alignment;

  /**
   * Version of the plan. Incremented when the plan is updated.
   */
  @summary("Version")
  @minValue(1)
  @visibility(Lifecycle.Read)
  version: integer = 1;

  /**
   * The currency code of the plan.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Currency")
  currency: CurrencyCode = "USD";

  /**
   * The date and time when the plan becomes effective. When not specified, the plan is a draft.
   */
  @visibility(Lifecycle.Read)
  @summary("Effective start date")
  effectiveFrom?: DateTime;

  /**
   * The date and time when the plan is no longer effective. When not specified, the plan is effective indefinitely.
   */
  @visibility(Lifecycle.Read)
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
  @visibility(Lifecycle.Read)
  status: PlanStatus;

  /**
   * The plan phase or pricing ramp allows changing a plan's rate cards over time as a subscription progresses.
   * A phase switch occurs only at the end of a billing period, ensuring that a single subscription invoice will not include charges from different phase prices.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
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
   * A semi-unique identifier for the resource.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Key")
  key: Key;

  /**
   * Human-readable name for the resource. Between 1 and 256 characters.
   */
  @summary("Display name")
  @minLength(1)
  @maxLength(256)
  name: string;

  /**
   * Optional description of the resource. Maximum 1024 characters.
   */
  @maxLength(1024)
  @summary("Description")
  description?: string;

  /**
   * Additional metadata for the resource.
   */
  @summary("Metadata")
  metadata?: Metadata | null;

  /**
   * The duration of the phase.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Duration")
  @encode(DurationKnownEncoding.ISO8601)
  @example(duration.fromISO("P1Y"))
  duration: duration | null;

  /**
   * The rate cards of the plan.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Rate cards")
  rateCards: RateCard[];

  /**
   * The discounts on the plan.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Discounts")
  discounts?: Discount[];

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
  /**
   * The currency code in ISO 4217 format.
   */
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

  /**
   * The discounts on the plan.
   */
  discounts?: Discount[];
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
