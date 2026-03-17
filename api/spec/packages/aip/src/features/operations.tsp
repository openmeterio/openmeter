import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../shared/index.tsp";
import "../meters/query.tsp";
import "./cost.tsp";
import "./feature.tsp";

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

  /**
   *  List all features.
   */
  @get
  @route("/features")
  @operationId("list-features")
  @summary("List features")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  list(
    ...Common.PagePaginationQuery,
  ): Shared.PagePaginatedResponse<Feature> | Common.ErrorResponses;

  /**
   * Create a feature.
   */
  @post
  @route("/features")
  @operationId("create-feature")
  @summary("Create a feature.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  create(
    @body
    feature: Shared.CreateRequest<Feature>,
  ): Shared.PagePaginatedResponse<Feature> | Common.ErrorResponses;

  /**
   * Get a feature by id.
   */
  @get
  @route("/features/{featureId}")
  @operationId("get-feature")
  @summary("Get a feature by id.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  get(@path featureId: Shared.ULID):
    | Shared.GetResponse<Feature>
    | Common.ErrorResponses
    | Common.Gone
    | Common.NotFound;

  /**
   * Delete a feautre by id.
   */
  @delete
  @route("/features/{featureId}")
  @operationId("delete-feature")
  @summary("Delete a feature by id.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  delete(
    @path featureId: Shared.ULID,
  ): Shared.DeleteResponse | Common.ErrorResponses | Common.NotFound;
}
