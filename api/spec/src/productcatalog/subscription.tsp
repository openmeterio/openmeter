import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "@typespec/versioning";

import "../types.tsp";
import "../errors.tsp";
import "./prices.tsp";
import "./ratecards.tsp";
import "./discounts.tsp";
import "../entitlements/main.tsp";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;
using TypeSpec.Versioning;

namespace OpenMeter.ProductCatalog;

/**
 * Expanded subscription
 */
@friendlyName("SubscriptionExpanded")
model SubscriptionExpanded {
  ...Subscription;
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
}

/**
 * Subscription is an exact subscription instance.
 */
@friendlyName("Subscription")
model Subscription {
  ...global.Resource;
  ...global.CadencedResource;

  /**
   * The status of the subscription.
   */
  @visibility("read")
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
}

/**
 * Expanded subscription phase
 */
@friendlyName("SubscriptionPhaseExpanded")
model SubscriptionPhaseExpanded {
  ...SubscriptionPhase;
  items: SubscriptionItem[];
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
  @example(duration.fromISO("P1Y1D"))
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
  discounts?: Discount[];

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
   * The billing cadence of the rate card.
   * When null, the rate card is a one-time purchase.
   */
  @summary("Billing cadence")
  @encode(DurationKnownEncoding.ISO8601)
  billingCandence: duration | null;

  /**
   * The price of the rate card.
   * When null, the feature or service is free.
   */
  @summary("Price")
  @example(#{ type: PriceType.flat, amount: "100", paymentTerm: "in_arrears" })
  price:
    | FlatPriceWithPaymentTerm
    | UnitPriceWithCommitments
    | TieredPriceWithCommitments
    | null;

  /**
   * Describes what access is gained via the SubscriptionItem
   */
  included?: SubscriptionItemIncluded;

  /**
   * The tax config of the Subscription Item.
   * When undefined, the tax config of the feature or the default tax config of the plan is used.
   */
  @visibility("read", "create", "update")
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

  /*
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
}

/**
 * Response body for subscription change.
 */
@friendlyName("SubscriptionChangeResponseBody")
model SubscriptionChangeResponseBody {
  @summary("Current subscription")
  current: Subscription;

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
  ...CustomSubscriptionChange;
  customerId: ULID;
}

/**
 * Change a custom subscription.
 */
@friendlyName("CustomSubscriptionChange")
model CustomSubscriptionChange {
  activeFrom: DateTime;
  customPlan: TypeSpec.Rest.Resource.ResourceCreateModel<Plan>;
}

/**
 * Create subscription based on plan.
 */
@friendlyName("PlanSubscriptionCreate")
model PlanSubscriptionCreate {
  ...PlanSubscriptionChange;
  customerId: ULID;
}

/**
 * Change subscription based on plan.
 */
@friendlyName("PlanSubscriptionChange")
model PlanSubscriptionChange {
  metadata?: global.Metadata;
  plan: PlanReferenceInput;
  activeFrom: DateTime;
  name: string;
  description?: string;
}

@friendlyName("SubscriptionEditOperation")
@oneOf
@discriminator("op")
union SubscriptionEditOperation {
  EditSubscriptionAddItem,
  EditSubscriptionRemoveItem,
  EditSubscriptionAddPhase,
  EditSubscriptionRemovePhase,
  EditSubscriptionStretchPhase,
}

@friendlyName("EditOp")
enum EditOp {
  AddItem: "add_item",
  RemoveItem: "remove_item",
  AddPhase: "add_phase",
  RemovePhase: "remove_phase",
  StretchPhase: "stretch_phase",
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
