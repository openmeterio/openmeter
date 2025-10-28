import "@typespec/http";
import "../types.tsp";
import "./ratecards.tsp";
import "./prorating.tsp";

namespace OpenMeter;

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
   * The default billing cadence for subscriptions using this plan.
   * Defines how often customers are billed using ISO8601 duration format.
   * Examples: "P1M" (monthly), "P3M" (quarterly), "P1Y" (annually).
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Billing cadence")
  @encode(DurationKnownEncoding.ISO8601)
  @example(duration.fromISO("P1M"))
  billingCadence: duration;

  /**
   * Default pro-rating configuration for subscriptions using this plan.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Pro-rating configuration")
  proRatingConfig?: ProRatingConfig = #{
    enabled: true,
    mode: ProRatingMode.proratePrices,
  };

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

  /**
   * List of validation errors.
   */
  @visibility(Lifecycle.Read)
  @summary("Validation errors")
  validationErrors: ValidationError[] | null;
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
}

/**
 * Order by options for plans.
 */
@friendlyName("PlanOrderBy")
enum PlanOrderBy {
  id: "id",
  key: "key",
  version: "version",
  createdAt: "created_at",
  updatedAt: "updated_at",
}
