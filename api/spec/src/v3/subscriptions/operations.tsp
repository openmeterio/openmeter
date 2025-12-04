import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Subscriptions;

/**
 * Request to create a customer subscription.
 */
@friendlyName("BillingCreateCustomerSubscriptionRequest")
model CreateCustomerSubscriptionRequest {
  /**
   * The plan to subscribe to.
   */
  @summary("Plan Reference")
  plan: PlanReference;
}

/**
 * Reference to a plan by its id or key and version.
 */
@oneOf
@friendlyName("BillingPlanReference")
union PlanReference {
  id: Shared.ULID,
  keyVersion: {
    key: string,
    version?: string,
  },
}

interface CustomerSubscriptionsOperations {
  @post
  @operationId("create-customer-subscription")
  @summary("Create customer subscription")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  create(
    @path customerId: Shared.ULID,
    @body request: CreateCustomerSubscriptionRequest,
  ): {
    @statusCode _: 204;
  } | Common.ErrorResponses;
}
