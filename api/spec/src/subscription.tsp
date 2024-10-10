import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "@typespec/versioning";

import "./types.tsp";
import "./errors.tsp";
import "./prices.tsp";
import "./ratecards.tsp";
import "./discounts.tsp";
import "./entitlements/main.tsp";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;
using TypeSpec.Versioning;

namespace OpenMeter;

@route("/api/v1/subscriptions")
@tag("Subscriptions")
interface Subscriptions {
  @get
  @operationId("getSubscription")
  @sharedRoute
  get(
    @path subscriptionId: ULID,
    @query expand?: false,

    /**
     * The time at which the subscription should be queried. If not provided the current time is used.
     */
    @query at?: DateTime,
  ): Subscription | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  @get
  @operationId("getSubscription")
  @sharedRoute
  getExpanded(
    @path subscriptionId: ULID,
    @query expand: true,

    /**
     * The time at which the subscription should be queried. If not provided the current time is used.
     */
    @query at?: DateTime,
  ): SubscriptionExpanded | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  create(
    @body body: {
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
    },
  ): Subscription | OpenMeter.ConflictError | OpenMeter.CommonErrors;

  /**
   * Batch processing commands for manipulating running subscriptions.
   * The key format is `/phases/{phaseKey}` or `/phases/{phaseKey}/items/{itemKey}`.
   *
   * Add operations insert a new member based on the creation input without altering the existing members.
   *
   * Remove operations remove the member from the collection / document.
   *
   * The extend operation extends the specific phase if possible, while delaying all subsequent phases by the same amount.
   */
  @patch
  @operationId("editSubscription")
  @route("/{subscriptionId}")
  edit(
    @header accept: "json-patch+json",
    @path subscriptionId: ULID,

    @body
    @maxItems(100)
    body: SubscriptionPatch[],
  ): Subscription | OpenMeter.ConflictError | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Migrates the subscripiton to the procided version of the plan.
   */
  @post
  @operationId("migrateSubscription")
  @route("/{subscriptionId}/migrate")
  migrate(
    @path subscriptionId: ULID,
    @body body: {
      @minValue(1)
      targetVersion: integer;
    },
  ): Subscription | OpenMeter.ConflictError | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Cancels the subscription.
   */
  @post
  @operationId("cancelSubscription")
  @route("/{subscriptionId}/cancel")
  cancel(
    @path subscriptionId: ULID,
    @body body: {
      /**
       * If not provided the subscription is canceled immediately.
       */
      effectiveDate?: DateTime;
    },
  ): Subscription | OpenMeter.ConflictError | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Cancels the scheduled cancelation.
   */
  @post
  @operationId("unscheduleCancelation")
  @route("/{subscriptionId}/unschedule-cancelation")
  unscheduleCancelation(@path subscriptionId: ULID):
    | Subscription
    | OpenMeter.ConflictError
    | OpenMeter.NotFoundError
    | OpenMeter.CommonErrors;
}

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
  plan: {
    /**
     * The plan Key of the subscription.
     */
    key: Key;

    /**
     * The plan version of the subscription.
     */
    version: integer;
  };

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
  ...OmitProperties<SubscriptionPhaseCreateInput, "startAfter">;

  /**
   * The time from which the phase is active.
   */
  activeFrom: DateTime;
}

@friendlyName("SubscriptionPhaseCreateInput")
model SubscriptionPhaseCreateInput {
  /**
   * Interval after the subscription starts to transition to the phase.
   * When null, the phase starts immediately after the subscription starts.
   */
  @summary("Start after")
  @encode(DurationKnownEncoding.ISO8601)
  startAfter: duration | null;

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
   * We say “referenced by the Price” regardless of how a price itself is referenced, it colloquially makes sense to say “paying the same price for the same thing”
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

    /**
     * The entitlement of the Subscription Item.
     */
    entitlement?: Entitlements.Entitlement;
  };
}

@pattern("^\\/phases\\/[a-zA-Z0-9\\-]+$")
scalar SubscriptionPhasePatchKey extends string;

alias SubscriptionPhasePatch = SubscriptionEditAdd<
  SubscriptionPhaseCreateInput,
  SubscriptionPhasePatchKey
> | SubscriptionPhaseExtend<
  {
    @encode(DurationKnownEncoding.ISO8601)
    extendBy: duration;
  },
  SubscriptionPhasePatchKey
> | SubscriptionEditRemove<SubscriptionPhasePatchKey>;

@pattern("^\\/phases\\/[a-zA-Z0-9\\-]+\\/items\\/[a-zA-Z0-9\\-]+$")
scalar SubscriptionItemPatchKey extends string;

alias SubscriptionPatch = SubscriptionItemPatch | SubscriptionPhasePatch;

alias SubscriptionItemPatch = SubscriptionEditAdd<
  RateCard,
  SubscriptionItemPatchKey
> | SubscriptionEditRemove<SubscriptionItemPatchKey>;

model SubscriptionEditAdd<T, K extends string> {
  `op`: "add";
  path: K;
  value: T;
}

model SubscriptionPhaseExtend<T, K extends string> {
  `op`: "extend";
  path: K;
  value: T;
}

model SubscriptionEditRemove<K extends string> {
  `op`: "remove";
  path: K;
}
