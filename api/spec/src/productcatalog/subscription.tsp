import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";

import "../types.tsp";
import "../errors.tsp";
import "./prices.tsp";
import "./ratecards.tsp";
import "./discounts.tsp";
import "../entitlements/main.tsp";

using TypeSpec.OpenAPI;

namespace OpenMeter.ProductCatalog;

/**
 * Expanded subscription
 */
@friendlyName("SubscriptionExpanded")
model SubscriptionExpanded {
  ...OmitProperties<Subscription, "alignment">;

  /**
   * Alignment details enriched with the current billing period.
   */
  alignment?: SubscriptionAlignment;

  /**
   * The phases of the subscription.
   */
  phases: SubscriptionPhaseExpanded[];
}

/**
 * Subscription status.
 */
@friendlyName("SubscriptionStatus")
enum SubscriptionStatus {
  Active: "active",
  Inactive: "inactive",
  Canceled: "canceled",
  Scheduled: "scheduled",
}

/**
 * Alignment details enriched with the current billing period.
 */
@friendlyName("SubscriptionAlignment")
model SubscriptionAlignment {
  ...Alignment;

  /**
   * The current billing period. Only has value if the subscription is aligned and active.
   */
  currentAlignedBillingPeriod?: Period;
}

/**
 * Subscription is an exact subscription instance.
 */
@friendlyName("Subscription")
model Subscription {
  ...global.Resource;
  ...global.CadencedResource;

  /**
   * Alignment configuration for the plan.
   */
  alignment?: Alignment;

  /**
   * The status of the subscription.
   */
  @visibility(Lifecycle.Read)
  status: SubscriptionStatus;

  /**
   * The customer ID of the subscription.
   */
  customerId: ULID;

  /**
   * The plan of the subscription.
   */
  plan?: PlanReference;

  /**
   * The currency code of the subscription.
   * Will be revised once we add multi currency support.
   */
  @summary("Currency")
  currency: CurrencyCode = "USD";

  /**
   * The billing cadence for the subscriptions.
   * Defines how often customers are billed using ISO8601 duration format.
   * Examples: "P1M" (monthly), "P3M" (quarterly), "P1Y" (annually).
   */
  @visibility(Lifecycle.Read)
  @summary("Billing cadence")
  @encode(DurationKnownEncoding.ISO8601)
  @example(duration.fromISO("P1M"))
  billingCadence: duration;

  /**
   * The pro-rating configuration for the subscriptions.
   */
  @visibility(Lifecycle.Read)
  @summary("Pro-rating configuration")
  proRatingConfig?: ProRatingConfig = #{
    enabled: true,
    mode: ProRatingMode.proratePrices,
  };

  /**
   * The normalizedbilling anchor of the subscription.
   */
  @visibility(Lifecycle.Read)
  @summary("Billing anchor")
  billingAnchor: DateTime;
}

/**
 * Expanded subscription phase
 */
@friendlyName("SubscriptionPhaseExpanded")
model SubscriptionPhaseExpanded {
  ...SubscriptionPhase;

  /**
   * The items of the phase. The structure is flattened to better conform to the Plan API.
   * The timelines are flattened according to the following rules:
   * - for the current phase, the `items` contains only the active item for each key
   * - for past phases, the `items` contains only the last item for each key
   * - for future phases, the `items` contains only the first version of the item for each key
   */
  items: SubscriptionItem[];

  /**
   * Includes all versions of the items on each key, including all edits, scheduled changes, etc...
   */
  itemTimelines: Record<SubscriptionItem[]>;
}

/**
 * Subscription phase, analogous to plan phases.
 */
@friendlyName("SubscriptionPhase")
model SubscriptionPhase {
  ...global.Resource;
  ...Keyed;

  // TODO: lets try this with visibility
  ...OmitProperties<
    SubscriptionPhaseCreate,
    "startAfter" | "duration" | "key" | "name" | "description"
  >;

  /**
   * The time from which the phase is active.
   */
  activeFrom: DateTime;

  /**
   * The until which the Phase is active.
   */
  activeTo?: DateTime;
}

/**
 * Subscription phase create input.
 */
@friendlyName("SubscriptionPhaseCreate")
model SubscriptionPhaseCreate {
  /**
   * Interval after the subscription starts to transition to the phase.
   * When null, the phase starts immediately after the subscription starts.
   */
  @summary("Start after")
  @encode(DurationKnownEncoding.ISO8601)
  @example(duration.fromISO("P1Y"))
  startAfter: duration | null;

  /**
   * The intended duration of the new phase.
   * Duration is required when the phase will not be the last phase.
   */
  @summary("Duration")
  @encode(DurationKnownEncoding.ISO8601)
  @example(duration.fromISO("P1M"))
  duration?: duration;

  /**
   * The discounts on the plan.
   */
  @summary("Discounts")
  discounts?: Discounts;

  /**
   * A locally unique identifier for the phase.
   */
  key: Key;

  /**
   * The name of the phase.
   */
  name: string;

  /**
   * The description of the phase.
   */
  description?: string;
}

/**
 * The actual contents of the Subscription, what the user gets, what they pay, etc...
 */
@friendlyName("SubscriptionItem")
model SubscriptionItem {
  ...global.Resource;
  ...global.CadencedResource;

  /**
   * The identifier of the RateCard.
   * SubscriptionItem/RateCard can be identified, it has a reference:
   *
   * 1. If a Feature is associated with the SubscriptionItem, it is identified by the Feature
   *  1.1 It can be an ID reference, for an exact version of the Feature (Features can change across versions)
   *  1.2 It can be a Key reference, which always refers to the latest (active or inactive) version of a Feature
   *
   * 2. If a Feature is not associated with the SubscriptionItem, it is referenced by the Price
   *
   * We say “referenced by the Price” regardless of how a price itself is referenced, it colloquially makes sense to say “paying the same price for the same thing”. In practice this should be derived from what's printed on the invoice line-item.
   */
  key: Key;

  /**
   * The feature's key (if present).
   */
  featureKey?: Key;

  /**
   * The billing cadence of the rate card.
   * When null, the rate card is a one-time purchase.
   */
  @summary("Billing cadence")
  @encode(DurationKnownEncoding.ISO8601)
  billingCadence: duration | null;

  /**
   * The price of the rate card.
   * When null, the feature or service is free.
   */
  @summary("Price")
  @example(#{ type: PriceType.flat, amount: "100", paymentTerm: "in_arrears" })
  price: RateCardUsageBasedPrice | null;

  /**
   * The discounts applied to the rate card.
   */
  @summary("Discounts")
  discounts?: Discounts;

  /**
   * Describes what access is gained via the SubscriptionItem
   */
  included?: SubscriptionItemIncluded;

  /**
   * The tax config of the Subscription Item.
   * When undefined, the tax config of the feature or the default tax config of the plan is used.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Tax config")
  taxConfig?: TaxConfig;
}

/**
 * Included contents like Entitlement, or the Feature.
 */
@friendlyName("SubscriptionItemIncluded")
model SubscriptionItemIncluded {
  /**
   * The feature the customer is entitled to use.
   */
  feature: Entitlements.Feature;

  /**
   * The entitlement of the Subscription Item.
   */
  entitlement?: Entitlements.Entitlement;
}

/**
 * Subscription edit input.
 */
@friendlyName("SubscriptionEdit")
model SubscriptionEdit {
  /**
   * Batch processing commands for manipulating running subscriptions.
   * The key format is `/phases/{phaseKey}` or `/phases/{phaseKey}/items/{itemKey}`.
   */
  @maxItems(100)
  customizations: SubscriptionEditOperation[];

  /**
   * Whether the billing period should be restarted.
   */
  // restartBillingPeriod?: boolean;
  /**
   * Timing configuration to allow for the changes to take effect at different times.
   */
  timing?: SubscriptionTiming;
}

/**
 * Response body for subscription change.
 */
@friendlyName("SubscriptionChangeResponseBody")
model SubscriptionChangeResponseBody {
  /**
   * The current subscription before the change.
   */
  @summary("Current subscription")
  current: Subscription;

  /**
   * The new state of the subscription after the change.
   */
  @summary("The subscription it will be changed to")
  next: SubscriptionExpanded;
}

/**
 * Create a subscription.
 */
@friendlyName("SubscriptionCreate")
@oneOf
union SubscriptionCreate {
  PlanSubscriptionCreate,
  CustomSubscriptionCreate,
}

/**
 * Change a subscription.
 */
@friendlyName("SubscriptionChange")
@oneOf
union SubscriptionChange {
  PlanSubscriptionChange,
  CustomSubscriptionChange,
}

/**
 * Create a custom subscription.
 */
@friendlyName("CustomSubscriptionCreate")
model CustomSubscriptionCreate {
  ...OmitProperties<CustomSubscriptionChange, "timing">;

  /**
   * Timing configuration for the change, when the change should take effect.
   * The default is immediate.
   */
  timing?: SubscriptionTiming = SubscriptionTimingEnum.Immediate;

  /**
   * The ID of the customer. Provide either the key or ID. Has presedence over the key.
   */
  customerId?: ULID;

  /**
   * The key of the customer. Provide either the key or ID.
   */
  customerKey?: ExternalKey;

  /**
   * The billing anchor of the subscription. The provided date will be normalized according to the billing cadence to the nearest recurrence before start time. If not provided, the subscription start time will be used.
   */
  billingAnchor?: DateTime;
}

/**
 * Change a custom subscription.
 */
@friendlyName("CustomSubscriptionChange")
model CustomSubscriptionChange {
  /**
   * Timing configuration for the change, when the change should take effect.
   * For changing a subscription, the accepted values depend on the subscription configuration.
   */
  timing: SubscriptionTiming;

  /**
   * The custom plan description which defines the Subscription.
   */
  customPlan: CustomPlanInput;
}

/**
 * Plan input for custom subscription creation (without key and version).
 */
@friendlyName("CustomPlanInput")
model CustomPlanInput
  extends OmitProperties<
    TypeSpec.Rest.Resource.ResourceCreateModel<Plan>,
    "key" | "version"
  > {}

/**
 * Create subscription based on plan.
 */
@friendlyName("PlanSubscriptionCreate")
model PlanSubscriptionCreate {
  ...OmitProperties<PlanSubscriptionChange, "timing">;

  /**
   * Timing configuration for the change, when the change should take effect.
   * The default is immediate.
   */
  timing?: SubscriptionTiming = SubscriptionTimingEnum.Immediate;

  /**
   * The ID of the customer. Provide either the key or ID. Has presedence over the key.
   */
  customerId?: ULID;

  /**
   * The key of the customer. Provide either the key or ID.
   */
  customerKey?: ExternalKey;

  /**
   * The billing anchor of the subscription. The provided date will be normalized according to the billing cadence to the nearest recurrence before start time. If not provided, the subscription start time will be used.
   */
  billingAnchor?: DateTime;
}

/**
 * Change subscription based on plan.
 */
@friendlyName("PlanSubscriptionChange")
model PlanSubscriptionChange {
  /**
   * Timing configuration for the change, when the change should take effect.
   * For changing a subscription, the accepted values depend on the subscription configuration.
   */
  timing: SubscriptionTiming;

  /**
   * What alignment settings the subscription should have.
   */
  alignment?: Alignment;

  /**
   * Arbitrary metadata associated with the subscription.
   */
  metadata?: global.Metadata;

  /**
   * The plan reference to change to.
   */
  plan: PlanReferenceInput;

  /**
   * The key of the phase to start the subscription in.
   * If not provided, the subscription will start in the first phase of the plan.
   */
  @minLength(1)
  startingPhase?: string;

  /**
   * The name of the Subscription. If not provided the plan name is used.
   */
  name?: string;

  /**
   * Description for the Subscription.
   */
  description?: string;
}

/**
 * Subscription edit timing defined when the changes should take effect.
 * If the provided configuration is not supported by the subscription, an error will be returned.
 */
@oneOf
@friendlyName("SubscriptionTiming")
union SubscriptionTiming {
  Enum: SubscriptionTimingEnum,
  Custom: DateTime,
}

/**
 * Subscription edit timing.
 * When immediate, the requested changes take effect immediately.
 * When nextBillingCycle, the requested changes take effect at the next billing cycle.
 */
@friendlyName("SubscriptionTimingEnum")
enum SubscriptionTimingEnum {
  Immediate: "immediate",
  NextBillingCycle: "next_billing_cycle",
}

/**
 * The operation to be performed on the subscription.
 */
@discriminated(#{ envelope: "none", discriminatorPropertyName: "op" })
@friendlyName("SubscriptionEditOperation")
union SubscriptionEditOperation {
  add_item: EditSubscriptionAddItem,
  remove_item: EditSubscriptionRemoveItem,
  add_phase: EditSubscriptionAddPhase,
  remove_phase: EditSubscriptionRemovePhase,
  stretch_phase: EditSubscriptionStretchPhase,
  unschedule_edit: EditSubscriptionUnscheduleEdit,
}

/**
 * Enum listing the different operation types.
 */
@friendlyName("EditOp")
enum EditOp {
  AddItem: "add_item",
  RemoveItem: "remove_item",
  UnscheduleEdit: "unschedule_edit",
  AddPhase: "add_phase",
  RemovePhase: "remove_phase",
  StretchPhase: "stretch_phase",
}

/**
 * Unschedules any edits from the current phase.
 */
@friendlyName("EditSubscriptionUnscheduleEdit")
model EditSubscriptionUnscheduleEdit {
  `op`: EditOp.UnscheduleEdit;
}

/**
 * Add a new item to a phase.
 */
@friendlyName("EditSubscriptionAddItem")
model EditSubscriptionAddItem {
  `op`: EditOp.AddItem;
  phaseKey: string;
  rateCard: RateCard;
}

/**
 * Remove an item from a phase.
 */
@friendlyName("EditSubscriptionRemoveItem")
model EditSubscriptionRemoveItem {
  `op`: EditOp.RemoveItem;
  phaseKey: string;
  itemKey: string;
}

/**
 * Add a new phase
 */
@friendlyName("EditSubscriptionAddPhase")
model EditSubscriptionAddPhase {
  `op`: EditOp.AddPhase;
  phase: SubscriptionPhaseCreate;
}

/**
 * Remove a phase
 */
@friendlyName("EditSubscriptionRemovePhase")
model EditSubscriptionRemovePhase {
  `op`: EditOp.RemovePhase;
  phaseKey: string;
  shift: RemovePhaseShifting;
}

/**
 * Stretch a phase
 */
@friendlyName("EditSubscriptionStretchPhase")
model EditSubscriptionStretchPhase {
  `op`: EditOp.StretchPhase;
  phaseKey: string;
  extendBy: duration;
}

/**
 * The direction of the phase shift when a phase is removed.
 */
@friendlyName("RemovePhaseShifting")
enum RemovePhaseShifting {
  /**
   * Shifts all subsequent phases to start sooner by the deleted phase's length
   */
  Next: "next",

  /**
   * Extends the previous phase to end later by the deleted phase's length
   */
  Prev: "prev",
}
