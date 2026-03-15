import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../shared/index.tsp";
import "../meters/query.tsp";
import "./cost.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Features;

interface FeatureCostOperations {
  /**
   * Query the cost of a feature.
   */
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @post
  @route("/query")
  @operationId("query-feature-cost")
  @summary("Query feature cost")
  queryCost(
    @path featureId: Shared.ULID,
    @body request?: Meters.MeterQueryRequest,
  ): FeatureCostQueryResult | Common.NotFound | Common.ErrorResponses;
}
