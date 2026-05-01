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
 * Request for canceling a subscription.
 */
@friendlyName("BillingSubscriptionCancel")
model SubscriptionCancel {
  /**
   * If not provided the subscription is canceled immediately.
   */
  timing?: SubscriptionEditTiming = SubscriptionEditTimingEnum.Immediate;
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

/**
 * Filter options for listing subscriptions.
 */
@friendlyName("ListSubscriptionsParamsFilter")
model ListSubscriptionsParamsFilter {
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  id?: Common.ULIDFieldFilter;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  customer_id?: Common.ULIDFieldFilter;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  status?: Common.StringFieldFilterExact;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  plan_id?: Common.ULIDFieldFilter;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  plan_key?: Common.StringFieldFilter;
}

interface SubscriptionsOperations {
  @post
  @operationId("create-subscription")
  @summary("Create subscription")
  create(@body subscription: SubscriptionCreate):
    | Shared.CreateResponse<Subscription>
    | Common.NotFound
    | Common.Conflict
    | Common.ErrorResponses;

  @get
  @operationId("list-subscriptions")
  @summary("List subscriptions")
  list(
    ...Common.PagePaginationQuery,

    /**
     * Sort subscriptions returned in the response.
     * Supported sort attributes are:
     * - `id`
     * - `active_from` (default)
     * - `active_to`
     */
    @query(#{ name: "sort" })
    sort?: Common.SortQuery,

    /**
     * Filter subscriptions.
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListSubscriptionsParamsFilter,
  ): Shared.PagePaginatedResponse<Subscription> | Common.NotFound | Common.ErrorResponses;

  @get
  @operationId("get-subscription")
  @summary("Get subscription")
  get(
    @path subscriptionId: Shared.ULID,
  ): Shared.GetResponse<Subscription> | Common.NotFound | Common.ErrorResponses;

  /**
   * Cancels the subscription.
   * Will result in a scheduling conflict if there are other subscriptions scheduled to start after the cancelation time.
   */
  @post
  @operationId("cancel-subscription")
  @summary("Cancel subscription")
  @route("/{subscriptionId}/cancel")
  cancel(@path subscriptionId: Shared.ULID, @body body: SubscriptionCancel):
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
  @route("/{subscriptionId}/unschedule-cancelation")
  unscheduleCancelation(@path subscriptionId: Shared.ULID):
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
  @route("/{subscriptionId}/change")
  change(
    @path subscriptionId: Shared.ULID,

    @body
    body: SubscriptionChange,
  ):
    | SubscriptionChangeResponse
    | Common.NotFound
    | Common.Conflict
    | Common.ErrorResponses;
}
