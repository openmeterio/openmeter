import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

import "./types.tsp";
import "./ratecards.tsp";
import "./plan.tsp";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter;

@route("/api/v1/subscriptions")
@tag("Subscriptions")
interface Subscriptions {
  @get
  @operationId("getSubscription")
  get(@path subscriptionId: ULID): Subscription | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  create(
    @body body: {
      customerId: ULID;
      templatingPlan: {
        planKey: Key;
        planVersion: integer;
      };
      currency: CurrencyCode;
      activeFrom: DateTime;
      overrides: unknown[]; // TBD
    },
  ): Subscription | OpenMeter.ConflictError | OpenMeter.CommonErrors;

  /**
   * Migrates the subscripiton to the procided version of the plan.
   */
  @post
  @operationId("migrateSubscription")
  @route("/{subscriptionId}/migrate")
  migrate(
    @path subscriptionId: ULID,
    @body body: {
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
      effective: "IMMEDIATE" | DateTime;
    },
  ): Subscription | OpenMeter.ConflictError | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Cancels the subscription.
   */
  @post
  @operationId("cancelScheduledCancelation")
  @route("/{subscriptionId}/cancel-scheduled-cancelation")
  cancelScheduledCancelation(@path subscriptionId: ULID):
    | Subscription
    | OpenMeter.ConflictError
    | OpenMeter.NotFoundError
    | OpenMeter.CommonErrors;

  @get
  @operationId("getSubscriptionItemsAt")
  @route("/{subscriptionId}/items")
  getItems(
    @path subscriptionId: ULID,
    @query at: DateTime,
  ): SubscriptionItem[] | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  @get
  @operationId("getPhases")
  @summary("Get the phases of a subscription")
  @route("/{subscriptionId}/phases")
  getPhases(@path subscriptionId: ULID): SubscriptionPhase[] | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  @delete
  @operationId("deleteSubscriptionPhase")
  @route("/{subscriptionId}/phases/{phaseKey}")
  deletePhase(@path subscriptionId: ULID, @path phaseKey: Key): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  @post
  @operationId("addSubscriptionPhase")
  @route("/{subscriptionId}/phases")
  addPhase(
    @path subscriptionId: ULID,
    @body phase: {
      key: Key;
      rateCards: RateCard[]; // TODO: we need common ground on this interface
      activeFrom: DateTime;
    },
  ): SubscriptionPhase | OpenMeter.ConflictError | OpenMeter.CommonErrors;

  /**
   * Extend has to delay both the current and all subsequent phases by the provided duration.
   */
  @post
  @operationId("extendSubscriptionPhase")
  @route("/{subscriptionId}/phases/{phaseKey}/extend")
  extendPhase(
    @path subscriptionId: ULID,
    @path phaseKey: Key,
    @body body: {
      @encode(DurationKnownEncoding.ISO8601)
      extendBy: duration;
    },
  ): SubscriptionPhase[] | OpenMeter.ConflictError | OpenMeter.CommonErrors;

  @post
  @operationId("addSubscriptionPhaseRateCard")
  @route("/{subscriptionId}/phases/{phaseKey}/items")
  addPhaseRateCard(
    @path subscriptionId: ULID,
    @path phaseKey: Key,
    @body rateCard: RateCard,
  ): SubscriptionItem | OpenMeter.ConflictError | OpenMeter.CommonErrors;

  @delete
  @operationId("deleteSubscriptionPhaseItem")
  @route("/{subscriptionId}/phases/{phaseKey}/items/{itemKey}")
  deletePhaseItem(@path subscriptionId: ULID, @path phaseKey: Key, @path itemKey: Key): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

/**
 * Subscription is an exact subscription instance.
 */
@friendlyName("Subscription")
model Subscription extends global.Resource {
  /**
   * The customer ID of the subscription.
   */
  customerId: ULID;

  /**
   * The templating plan of the subscription.
   */
  templatingPlan: {
    /**
     * The plan Key of the subscription.
     */
    planKey: Key;

    /**
     * The plan version of the subscription.
     */
    planVersion: integer;
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
 * Subscription phase, analogous to plan phases.
 */
@friendlyName("SubscriptionPhase")
model SubscriptionPhase {
  /**
   * The time at which the phase becomes active.
   */
  activeFrom: DateTime;

  /**
   * The templating plan's phase which templated this phase.
   *
   * Undefined if the phase is not templated, i.e. manually created.
   */
  templatingPlanPhase?: Key;

  /**
   * The key by which the phase can be identified.
   */
  key: Key;

  /**
   * The contents of the phase
   */
  @summary("Items")
  items: SubscriptionItem[];

  /**
   * The discount on the plan.
   */
  @summary("Discount")
  discount?: AppliedDiscount;
}

@friendlyName("SubscriptionItem")
model SubscriptionItem {
  /**
   * Copied from product doc:
   *
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
   *
   * In practice this will need an ID being generated
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
   * Access describes what access is gained via the SubscriptionItem
   */
  feature?: {
    /**
     * The feature the customer is entitled to use.
     */
    featureKey: Key;

    /**
     * The entitlement of the Subscription Item.
     */
    entitlement?: Entitlements.Entitlement;
  };
}
