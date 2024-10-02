import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter.Entitlements;

/**
 * Order by options for features.
 */
@friendlyName("FeatureOrderBy")
enum FeatureOrderBy {
  ID: "id",
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  CreatedAt: "createdAt",
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  UpdatedAt: "updatedAt",
}

@route("/api/v1/features")
@tag("Entitlements (Experimental)")
interface Features {
  /**
   * List features.
   */
  @get
  @operationId("listFeatures")
  list(
    /**
     * Filter by meterSlug
     */
    @query meterSlug?: string,

    /**
     * Filter by meterGroupByFilters
     */
    @query includeArchived?: boolean = false,

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryLimitOffset,
    ...OpenMeter.QueryOrdering<FeatureOrderBy>,
  ): Feature[] | OpenMeter.PaginatedResponse<Feature> | OpenMeter.CommonErrors;

  /**
   * Features are either metered or static. A feature is metered if meterSlug is provided at creation.
   * For metered features you can pass additional filters that will be applied when calculating feature usage, based on the meter's groupBy fields.
   * Only meters with SUM and COUNT aggregation are supported for features.
   * Features cannot be updated later, only archived.
   */
  @post
  @operationId("createFeature")
  create(@body feature: FeatureCreateInputs): {
    @statusCode _: 201;
    @body body: Feature;
  } | OpenMeter.CommonErrors;

  /**
   * Get a feature by ID.
   */
  @get
  @operationId("getFeature")
  get(@path featureId: string): Feature | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Archive a feature by ID.
   *
   * Once a feature is archived it cannot be unarchived. If a feature is archived, new entitlements cannot be created for it, but archiving the feature does not affect existing entitlements.
   * This means, if you want to create a new feature with the same key, and then create entitlements for it, the previous entitlements have to be deleted first on a per subject basis.
   */
  @delete
  @operationId("deleteFeature")
  delete(@path featureId: string): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

/**
 * Represents a feature that can be enabled or disabled for a plan.
 * Used both for product catalog and entitlements.
 */
@friendlyName("FeatureCreateInputs")
model FeatureCreateInputs {
  @summary("The unique key of the feature")
  key: Key;

  @summary("The human-readable name of the feature")
  name: string;

  @summary("Optional metadata")
  @example(#{ key: "value" })
  metadata?: Metadata;

  // /**
  //  * The meter that the feature is associated with and and based on which usage is calculated.
  //  * The meter selected must have SUM or COUNT aggregation.
  //  */
  // @summary("Meter key")
  // @example("tokens_total")
  // meterKey?: Key;

  @summary("Meter slug")
  @example("tokens_total")
  meterSlug?: Key;

  /**
   * Optional meter group by filters.
   * Useful if the meter scope is broader than what feature tracks.
   * Example scenario would be a meter tracking all token use with groupBy fields for the model,
   * then the feature could filter for model=gpt-4.
   */
  @summary("Meter group by filters")
  @example(#{ `model`: "gpt-4", type: "input" })
  meterGroupByFilters?: Record<string>;
}

/**
 * Represents a feature that can be enabled or disabled for a plan.
 * Used both for product catalog and entitlements.
 */
@friendlyName("Feature")
model Feature {
  ...ResourceTimestamps;
  ...Archiveable;
  ...FeatureCreateInputs;
}
