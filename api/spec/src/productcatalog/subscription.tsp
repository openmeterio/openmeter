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
 * Subscription is an exact subscription instance.
 */
@friendlyName("Subscription")
model Subscription {
  ...global.Resource;

  /**
   * The customer ID of the subscription.
   */
  customerId: ULID;

  /**
   * The base plan of the subscription.
   */
  plan: PlanReference;

  /**
   * The currency code of the subscription.
   * Will be revised once we add multi currency support.
   */
  @summary("Currency")
  currency: CurrencyCode = "USD";

  /**
   * If the subscription is canceled or otherwise have to end activeTo denotes the end date.
   */
  activeTo?: DateTime;
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
  ...VirtualResource;
  ...OmitProperties<SubscriptionPhaseCreate, "startAfter">;

  /**
   * The time from which the phase is active.
   */
  activeFrom: DateTime;
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
   */
  @summary("Duration")
  @encode(DurationKnownEncoding.ISO8601)
  @example(duration.fromISO("P1M"))
  duration: duration;

  /**
   * The discounts on the plan.
   */
  @summary("Discount")
  discounts?: AppliedDiscount[];
}

/**
 * The actual contents of the Subscription, what the user gets, what they pay, etc...
 */
@friendlyName("SubscriptionItem")
model SubscriptionItem {
  // Key is overriden to add extra doc. Uses the same type.
  ...OmitProperties<VirtualResource, "key">;

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
    | FlatPriceWithPaymentTerm
    | null;

  /**
   * Describes what access is gained via the SubscriptionItem
   */
  included?: {
    /**
     * The feature the customer is entitled to use.
     */
    feature: Entitlements.Feature;

    /*
     * The entitlement of the Subscription Item.
     */
    entitlement?: Entitlements.Entitlement;
  };

  /**
   * The tax config of the Subscription Item.
   * When undefined, the tax config of the feature or the default tax config of the plan is used.
   */
  @visibility("read", "create", "update")
  @summary("Tax config")
  taxConfig?: TaxConfig;
}

/**
 * Create subscription request body.
 */
@friendlyName("CreateSubscriptionRequestBody")
model CreateSubscriptionRequestBody {
  customerId: ULID;
  plan: {
    key: Key;
    version: integer;
  };
  currency: CurrencyCode;
  activeFrom: DateTime;

  /**
   * Batch processing commands for customizing the susbcription.
   * The key format is `/phases/{phaseKey}/items/{itemKey}`.
   *
   * Add operations insert a new member based on the creation input without altering the existing members.
   *
   * Remove operations remove the member from the document.
   *
   */
  @maxItems(100)
  customizations: SubscriptionItemPatch[];
}

@pattern("^\\/phases\\/[a-zA-Z0-9\\-]+$")
scalar SubscriptionPhasePatchKey extends string;

/**
 * The direction of the phase shift when a phase is removed.
 */
@friendlyName("RemovePhaseShifting")
enum RemovePhaseShifting {
  /**
   * Shifts all subsequent phases to start sonner by the deleted phase's length
   */
  Next,

  /**
   * Extends the previous phase to end later by the deleted phase's length
   */
  Prev,
}

alias SubscriptionPhasePatch = SubscriptionEditAdd<
  SubscriptionPhaseCreate,
  SubscriptionPhasePatchKey
> | SubscriptionPhaseExtend<
  {
    @encode(DurationKnownEncoding.ISO8601)
    extendBy: duration;
  },
  SubscriptionPhasePatchKey
> | SubscriptionEditRemoveWithValue<
  {
    shift: RemovePhaseShifting;
  },
  SubscriptionPhasePatchKey
>;

@pattern("^\\/phases\\/[a-zA-Z0-9\\-]+\\/items\\/[a-zA-Z0-9\\-]+$")
scalar SubscriptionItemPatchKey extends string;

alias SubscriptionPatch = SubscriptionItemPatch | SubscriptionPhasePatch;

alias SubscriptionItemPatch = SubscriptionEditAdd<
  RateCard,
  SubscriptionItemPatchKey
> | SubscriptionEditRemove<SubscriptionItemPatchKey>;

/**
 * Subscription item add operation.
 */
@friendlyName("SubscriptionEditAdd")
model SubscriptionEditAdd<T, K extends string> {
  `op`: "add";
  path: K;
  value: T;
}

/**
 * Subscription phase extend operation.
 */
@friendlyName("SubscriptionEditExtend")
model SubscriptionPhaseExtend<T, K extends string> {
  `op`: "extend";
  path: K;
  value: T;
}

/**
 * Subscription phase remove operation.
 */
@friendlyName("SubscriptionEditRemove")
model SubscriptionEditRemove<K extends string> {
  `op`: "remove";
  path: K;
}

/**
 * Subscription item remove operation with a value.
 */
@friendlyName("SubscriptionEditRemoveWithValue")
model SubscriptionEditRemoveWithValue<T, K extends string> {
  `op`: "remove";
  path: K;
  value: T;
}
