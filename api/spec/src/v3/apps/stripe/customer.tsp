import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";
import "../../shared/index.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Apps.Stripe;

interface CustomerStripeEndpoints {
  // /**
  //  * Get stripe app data for a customer.
  //  * Only returns data if the customer billing profile is linked to a stripe app.
  //  */
  // @get
  // @operationId("getCustomerStripeAppData")
  // @summary("Get customer stripe app data")
  // get(
  //   @path customerIdOrKey: Shared.ULID | Shared.ExternalResourceKey,
  // ): Shared.GetResponse<StripeCustomerAppData> | Common.NotFound | Common.ErrorResponses;
}
