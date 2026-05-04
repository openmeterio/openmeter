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

/**
 * Filter options for listing features.
 */
@friendlyName("ListFeatureParamsFilter")
model ListFeaturesParamsFilter {
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  meter_id?: Common.ULIDFieldFilter;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  key?: Common.StringFieldFilter;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  name?: Common.StringFieldFilter;
}

interface FeatureOperations {
  /**
   *  List all features.
   */
  @get
  @operationId("list-features")
  @summary("List features")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.PrivateExtension, true)
  list(
    ...Common.PagePaginationQuery,

    /**
     * Sort features returned in the response.
     * Supported sort attributes are:
     * - `key`
     * - `name`
     * - `created_at` (default)
     * - `updated_at`
     *
     * The `asc` suffix is optional as the default sort order is ascending.
     * The `desc` suffix is used to specify a descending order.
     */
    @query(#{ name: "sort" })
    sort?: Common.SortQuery,

    /**
     * Filter features returned in the response.
     *
     * To filter features by meter_id add the following query param: filter[meter_id][oeq]=<id>
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListFeaturesParamsFilter,
  ): Shared.PagePaginatedResponse<Feature> | Common.ErrorResponses;

  /**
   * Create a feature.
   */
  @post
  @operationId("create-feature")
  @summary("Create feature")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.PrivateExtension, true)
  create(
    @body
    feature: Shared.CreateRequest<Feature>,
  ): Shared.CreateResponse<Feature> | Common.ErrorResponses;

  /**
   * Get a feature by id.
   */
  @get
  @route("/{featureId}")
  @operationId("get-feature")
  @summary("Get feature")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.PrivateExtension, true)
  get(@path featureId: Shared.ULID):
    | Shared.GetResponse<Feature>
    | Common.ErrorResponses
    | Common.Gone
    | Common.NotFound;

  /**
   * Update a feature by id.
   * Currently only the unit_cost field can be updated.
   */
  @patch
  @route("/{featureId}")
  @operationId("update-feature")
  @summary("Update feature")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.PrivateExtension, true)
  update(
    @path featureId: Shared.ULID,
    @body feature: FeatureUpdateRequest,
  ): Shared.UpdateResponse<Feature> | Common.ErrorResponses | Common.NotFound;

  /**
   * Delete a feature by id.
   */
  @delete
  @route("/{featureId}")
  @operationId("delete-feature")
  @summary("Delete feature")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.PrivateExtension, true)
  delete(
    @path featureId: Shared.ULID,
  ): Shared.DeleteResponse | Common.ErrorResponses | Common.NotFound;
}

interface FeatureCostOperations {
  /**
   * Query the cost of a feature.
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.PrivateExtension, true)
  @post
  @route("/query")
  @operationId("query-feature-cost")
  @summary("Query feature cost")
  queryCost(
    @path featureId: Shared.ULID,
    @body request?: Meters.MeterQueryRequest,
  ): FeatureCostQueryResult | Common.NotFound | Common.ErrorResponses;
}
