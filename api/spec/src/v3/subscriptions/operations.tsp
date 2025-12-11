import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./subscription.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Subscriptions;

interface CustomerSubscriptionsOperations {
  @post
  @operationId("create-customer-subscription")
  @summary("Create customer subscription")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  create(
    @path customerId: Shared.ULID,
    @body request: Shared.CreateRequest<Subscription>,
  ): Shared.CreateResponse<Subscription> | Common.NotFound | Common.Conflict | Common.ErrorResponses;

  @get
  @operationId("list-customer-subscriptions")
  @summary("List customer subscriptions")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  list(
    @path customerId: Shared.ULID,
    ...Common.PagePaginationQuery,
  ): Shared.PagePaginatedResponse<Subscription> | Common.NotFound | Common.ErrorResponses;
}
