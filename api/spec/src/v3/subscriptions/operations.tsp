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

interface SubscriptionsOperations {
  @post
  @operationId("create-subscription")
  @summary("Create subscription")
  create(
    @path customerId: Shared.ULID,
    @body request: Shared.CreateRequest<Subscription>,
  ):
    | Shared.CreateResponse<Subscription>
    | Common.NotFound
    | Common.Conflict
    | Common.ErrorResponses;

  @get
  @operationId("list-subscriptions")
  @summary("List subscriptions")
  list(
    ...ListSubscriptionsParams,
  ): Shared.PagePaginatedResponse<Subscription> | Common.NotFound | Common.ErrorResponses;

  @get
  @operationId("get-subscription")
  @summary("Get subscription")
  get(
    @path subscriptionId: Shared.ULID,
  ): Shared.GetResponse<Subscription> | Common.NotFound | Common.ErrorResponses;
}
