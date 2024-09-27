namespace OpenMeter.ProductCatalog;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@route("/api/v1/plans")
@tag("Plan (Experimental)")
interface PlansById {
  /**
   * List all plans.
   */
  @get
  @operationId("listPlans")
  @summary("List plans")
  list(
    /**
     * Include deleted plans in response.
     *
     * Usage: `?includeDeleted=true`
     */
    @query
    @example(true)
    includeDeleted?: boolean = false,

    /**
     * Filter by plan.id attribute
     */
    @query(#{ explode: true })
    id?: string[],

    /**
     * Filter by plan.key attribute
     */
    @query(#{ explode: true })
    key?: string[],

    /**
     * Filter by plan.key and plan.version attributes
     */
    @query(#{ explode: true, style: "deepObject" })
    keyVersion?: Record<integer>,

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryOrdering<PlanOrderBy>,
  ): OpenMeter.PaginatedResponse<Plan> | OpenMeter.CommonErrors;

  /**
   * Create a new plan.
   */
  @post
  @operationId("createPlan")
  @summary("Create a plan")
  create(@body request: TypeSpec.Rest.Resource.ResourceCreateModel<Plan>): {
    @statusCode _: 201;
    @bodyRoot body: Plan;
  }  | OpenMeter.CommonErrors;

  /**
   * Update plan.
   */
  @put
  @operationId("updatePlan")
  @summary("Update a plan")
  update(
    @path planId: ULID,
    @body request: TypeSpec.Rest.Resource.ResourceCreateOrUpdateModel<Plan>,
  ): {
    @statusCode _: 200;
    @bodyRoot body: Plan;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Get a plan by id.
   */
  @get
  @operationId("getPlan")
  @summary("Get notification channel")
  get(@path planId: ULID): Plan | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Soft delete plan by id.
   *
   * Once a plan is deleted it cannot be undeleted.
   */
  @delete
  @operationId("deletePlan")
  @summary("Delete a notification channel")
  delete(@path planId: ULID): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

@route("/api/v1/plans/{planId}/publish")
@tag("Plan (Experimental)")
interface PlanPublishById {
  /**
   * Publish a new plan.
   */
  @post
  @operationId("publishPlanById")
  @summary("Publish plan")
  publish(@path planId: ULID): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

@route("/api/v1/plans/{planId}/unpublish")
@tag("Plan (Experimental)")
interface PlanUnpublishById {
  /**
   * Unpublish a new plan.
   */
  @post
  @operationId("unpublishPlanById")
  @summary("Unpublish plan")
  unpublish(@path planId: ULID): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

@route("/api/v1/plans/{planKey}/versions")
@tag("Plan (Experimental)")
interface PlansByKeyVersion {
  /**
   * List all version of a plan.
   */
  @get
  @operationId("listPlansByKeyVersion")
  @summary("List plans")
  list(
    @path planKey: Key,
    /**
     * Include deleted plans in response.
     *
     * Usage: `?includeDeleted=true`
     */
    @query
    @example(true)
    includeDeleted?: boolean = false,

    /**
     * Filter by plan.version attribute
     */
    @query(#{ explode: true })
    version?: integer[],

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryOrdering<PlanOrderBy>,
  ): OpenMeter.PaginatedResponse<Plan> | OpenMeter.CommonErrors;

  /**
   * Update plan.
   */
  @put
  @operationId("updatePlanByKeyVersion")
  @summary("Update plan")
  update(
    @path planKey: Key,
    @path planVersion: integer,
    @body request: TypeSpec.Rest.Resource.ResourceCreateOrUpdateModel<Plan>,
  ): {
    @statusCode _: 200;
    @bodyRoot body: Plan;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Get a plan by id.
   */
  @get
  @operationId("getPlanByKeyVersion")
  @summary("Get plan")
  get(
    @path planKey: Key,
    @path planVersion: integer,
  ): Plan | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Delete plan by id.
   *
   * Once a plan is deleted it cannot be undeleted.
   */
  @delete
  @operationId("deletePlanByKeyVersion")
  @summary("Delete plan")
  delete(
    @path planKey: Key,
    @path planVersion: integer,
  ): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

@route("/api/v1/plans/{planKey}/versions/{planVersion}/publish")
@tag("Plan (Experimental)")
interface PlanPublishByKeyVersion {
  /**
   * Publish a new plan.
   */
  @post
  @operationId("publishPlanByKeyVersion")
  @summary("Publish plan")
  publish(
    @path planKey: Key,
    @path planVersion: integer,
  ): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

@route("/api/v1/plans/{planKey}/versions/{planVersion}/unpublish")
@tag("Plan (Experimental)")
interface PlanUnpublishByKeyVersion {
  /**
   * Unpublish a new plan.
   */
  @post
  @operationId("unpublishPlanByKeyVersion")
  @summary("Unpublish plan")
  unpublish(
    @path planKey: Key,
    @path planVersion: integer,
  ): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

@route("/api/v1/plans/{planId}/phases")
@tag("Plan (Experimental)")
interface PlanByIdPhases {
  /**
   * List all phases in plan.
   */
  @get
  @operationId("listPlanByIdPhases")
  @summary("List phases in plan")
  list(
    @path planId: ULID,

    /**
     * Filter by phase.key attribute
     */
    @query(#{ explode: true })
    key?: string[],

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryOrdering<PhasesOrderBy>,
  ): OpenMeter.PaginatedResponse<PlanPhase> | OpenMeter.CommonErrors;

  /**
   * Create new phase in plan.
   */
  @post
  @operationId("createPlanByIdPhases")
  @summary("Create new phase in plan")
  create(
    @path planId: ULID,
    @body request: TypeSpec.Rest.Resource.ResourceCreateModel<PlanPhase>,
  ): {
    @statusCode _: 201;
    @bodyRoot body: PlanPhase;
  } | OpenMeter.CommonErrors;

  /**
   * Update phase in plan.
   */
  @put
  @operationId("updatePlanByIdPhases")
  @summary("Update phase in plan")
  update(
    @path planId: ULID,
    @path planPhaseKey: Key,
    @body request: TypeSpec.Rest.Resource.ResourceCreateOrUpdateModel<PlanPhase>,
  ): {
    @statusCode _: 200;
    @bodyRoot body: PlanPhase;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Get phase in plan.
   */
  @get
  @operationId("getPlanByIdPhases")
  @summary("Get phase in plan")
  get(
    @path planId: ULID,
    @path planPhaseKey: Key,
  ): PlanPhase | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Delete phase in plan.
   *
   * Once a phase is deleted it cannot be undeleted.
   */
  @delete
  @operationId("deletePlanByIdPhases")
  @summary("Delete phase in plan")
  delete(
    @path planId: ULID,
    @path planPhaseKey: Key,
  ): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

@route("/api/v1/plans/{planKey}/versions/{planVersion}/phases")
@tag("Plan (Experimental)")
interface PlanByKeyVersionPhases {
  /**
   * List all phases in plan.
   */
  @get
  @operationId("listPlanByKeyVersionPhases")
  @summary("List plans")
  list(
    @path planKey: Key,
    @path planVersion: integer,

    /**
     * Filter by phase.key attribute
     */
    @query(#{ explode: true })
    key?: string[],

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryOrdering<PhasesOrderBy>,
  ): OpenMeter.PaginatedResponse<Plan> | OpenMeter.CommonErrors;

  /**
   * Create new phase in plan.
   */
  @post
  @operationId("createPlanByKeyVersionPhases")
  @summary("Create new phase in plan")
  create(
    @path planKey: Key,
    @path planVersion: integer,
    @body request: TypeSpec.Rest.Resource.ResourceCreateModel<PlanPhase>,
  ): {
    @statusCode _: 201;
    @bodyRoot body: PlanPhase;
  } | OpenMeter.CommonErrors;

  /**
   * Create new phase in plan.
   */
  @put
  @operationId("updatePlanByKeyVersionPhases")
  @summary("Update plan")
  update(
    @path planKey: Key,
    @path planVersion: integer,
    @path phaseKey: Key,
    @body request: TypeSpec.Rest.Resource.ResourceCreateOrUpdateModel<PlanPhase>,
  ): {
    @statusCode _: 200;
    @bodyRoot body: PlanPhase;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Get phase in plan.
   */
  @get
  @operationId("getPlanByKeyVersionPhases")
  @summary("Get plan")
  get(
    @path planKey: Key,
    @path planVersion: integer,
    @path phaseKey: Key,
  ): PlanPhase | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Delete phase in plan.
   *
   * Once a phase is deleted it cannot be undeleted.
   */
  @delete
  @operationId("deletePlanByKeyVersionPhases")
  @summary("Delete plan")
  delete(
    @path planKey: Key,
    @path planVersion: integer,
  ): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}
