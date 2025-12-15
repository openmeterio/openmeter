import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../shared/index.tsp";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "./subscription.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Subscriptions;

/**
 * Query params for listing subscriptions.
 */
@friendlyName("ListSubscriptionsParams")
model ListSubscriptionsParams {
  ...Common.PagePaginationQuery;

  /**
   * Sort subscriptions returned in the response.
   * Supported sort attributes are:
   * - `id`
   * - `created_at`
   *
   * The `asc` suffix is optional as the default sort order is ascending.
   * The `desc` suffix is used to specify a descending order.
   */
  @query(#{ name: "sort" })
  sort?: Common.SortQuery;

  /**
   * Filter subscriptions.
   */
  @query(#{ style: "deepObject", explode: true })
  filter?: {
    /**
     * Filter subscriptions by customer ID.
     */
    customer_id?: Shared.ULID;
  };
}

/**
 * Request for canceling a subscription.
 */
@friendlyName("BillingSubscriptionCancel")
model SubscriptionCancel {
  /**
   * If not provided the subscription is canceled immediately.
   */
  timing?: SubscriptionEditTiming = SubscriptionEditTimingEnum.immediate;
}

/**
 * Request for changing a subscription.
 */
@friendlyName("BillingSubscriptionChange")
model SubscriptionChange {
  ...OmitProperties<SubscriptionCreate, "customer_id" | "customer_key">;

  /**
   * Timing configuration for the change, when the change should take effect.
   * For changing a subscription, the accepted values depend on the subscription configuration.
   */
  timing: SubscriptionEditTiming;
}

/**
 * Response for changing a subscription.
 */
@friendlyName("BillingSubscriptionChangeResponse")
model SubscriptionChangeResponse {
  /**
   * The current subscription before the change.
   */
  current: Subscription;

  /**
   * The new state of the subscription after the change.
   */
  next: Subscription;
}

interface SubscriptionsOperations {
  @post
  @operationId("create-subscription")
  @summary("Create subscription")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  create(@body subscription: SubscriptionCreate):
    | Shared.CreateResponse<Subscription>
    | Common.NotFound
    | Common.Conflict
    | Common.ErrorResponses;

  @get
  @operationId("list-subscriptions")
  @summary("List subscriptions")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  list(
    ...ListSubscriptionsParams,
  ): Shared.PagePaginatedResponse<Subscription> | Common.NotFound | Common.ErrorResponses;

  @get
  @operationId("get-subscription")
  @summary("Get subscription")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  get(
    @path subscriptionId: Shared.ULID,
  ): Shared.GetResponse<Subscription> | Common.NotFound | Common.ErrorResponses;

  /**
   * Cancels the subscription.
   * Will result in a scheduling conflict if there are other subscriptions scheduled to start after the cancellation time.
   */
  @post
  @operationId("cancel-subscription")
  @summary("Cancel subscription")
  @route("/cancel")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  cancel(@path subscriptionId: ULID, @body body: SubscriptionCancel):
    | Shared.UpdateResponse<Subscription>
    | Common.NotFound
    | Common.Conflict
    | Common.ErrorResponses;

  /**
   * Unschedules the subscription cancelation.
   */
  @post
  @operationId("unschedule-cancelation")
  @summary("Unschedule subscription cancelation")
  @route("/unschedule-cancelation")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  unscheduleCancelation(@path subscriptionId: ULID):
    | Shared.UpdateResponse<Subscription>
    | Common.NotFound
    | Common.Conflict
    | Common.ErrorResponses;

  /**
   * Closes a running subscription and starts a new one according to the specification.
   * Can be used for upgrades, downgrades, and plan changes.
   */
  @post
  @operationId("change-subscription")
  @summary("Change subscription")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @route("/change")
  change(
    @path subscriptionId: ULID,

    @body
    body: SubscriptionChange,
  ):
    | SubscriptionChangeResponse
    | Common.NotFound
    | Common.Conflict
    | Common.ErrorResponses;
}
