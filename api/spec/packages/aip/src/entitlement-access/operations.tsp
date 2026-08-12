import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../shared/index.tsp";
import "./entitlement-access.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace EntitlementAccess;

interface EntitlementAccessOperations {
  /**
   * Query feature access for a list of customers.
   *
   * The endpoint resolves each provided identifier to a customer and returns the
   * access status for the requested features, plus optional credit balance
   * availability.
   *
   * _Designed to be called on a fixed refresh interval and the query response is
   * intended to be cached._
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  @post
  @route("/query")
  @operationId("query-entitlement-access")
  @summary("Query entitlement access")
  query(
    ...Common.CursorPaginationQuery,
    @body _: EntitlementAccessQueryRequest,
  ): EntitlementAccessQueryResponse | Common.ErrorResponses;
}
