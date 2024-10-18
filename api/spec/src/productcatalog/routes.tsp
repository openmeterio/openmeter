namespace OpenMeter.ProductCatalog;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

alias PlanIdOrKey = ULID | Key;

@route("/api/v1/plans")
@tag("Plan (Experimental)")
interface Plans {
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
    id?: ULID[],

    /**
     * Filter by plan.key attribute
     */
    @query(#{ explode: true })
    key?: Key[],

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
  } | OpenMeter.CommonErrors;

  /**
   * Update plan by id.
   */
  @put
  @operationId("updatePlan")
  @summary("Update a plan")
  update(
    @path planId: ULID,
    @body request: TypeSpec.Rest.Resource.ResourceCreateOrUpdateModel<Plan>,
  ): Plan | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Get a plan by id or key. The latest published version is returned if latter is used.
   */
  @get
  @operationId("getPlan")
  @summary("Get plan")
  get(
    @extension("x-go-type", "string")
    @path
    planId: PlanIdOrKey,
  ): Plan | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Soft delete plan by plan.id.
   *
   * Once a plan is deleted it cannot be undeleted.
   */
  @delete
  @operationId("deletePlan")
  @summary("Delete plan")
  delete(@path planId: ULID): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

@route("/api/v1/plans/{planId}/publish")
@tag("Plan (Experimental)")
interface PlanPublish {
  /**
   * Publish a new plan.
   */
  @post
  @operationId("publishPlan")
  @summary("Publish plan")
  publish(@path planId: ULID): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

@route("/api/v1/plans/{planId}/unpublish")
@tag("Plan (Experimental)")
interface PlanUnpublish {
  /**
   * Unpublish a new plan.
   */
  @post
  @operationId("unpublishPlan")
  @summary("Unpublish plan")
  unpublish(@path planId: ULID): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

@route("/api/v1/plans/{planId}/next")
@tag("Plan (Experimental)")
interface PlanNewDraftVersion {
  /**
   * Create a new draft version from plan.
   * It returns error if there is already a plan in draft or planId does not reference the latest published version.
   */
  @post
  @operationId("newDraftPlan")
  @summary("New draft plan")
  next(
    @extension("x-go-type", "string")
    @path
    planId: PlanIdOrKey,
  ): {
    @statusCode _: 201;
    @bodyRoot body: Plan;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

@route("/api/v1/plans/{planId}/phases")
@tag("Plan (Experimental)")
interface PlanPhases {
  /**
   * List all phases in plan.
   */
  @get
  @operationId("listPlanPhases")
  @summary("List phases in plan")
  list(
    @path planId: ULID,

    /**
     * Filter by phase.key attribute
     */
    @query(#{ explode: true })
    key?: Key[],

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryOrdering<PhasesOrderBy>,
  ): OpenMeter.PaginatedResponse<PlanPhase> | OpenMeter.CommonErrors;

  /**
   * Create new phase in plan.
   */
  @post
  @operationId("createPlanPhases")
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
  @operationId("updatePlanPhases")
  @summary("Update phase in plan")
  update(
    @path planId: ULID,
    @path planPhaseKey: Key,

    @body
    request: TypeSpec.Rest.Resource.ResourceCreateOrUpdateModel<PlanPhase>,
  ): PlanPhase | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Get phase in plan.
   */
  @get
  @operationId("getPlanPhases")
  @summary("Get phase for plan")
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
  @operationId("deletePlanPhases")
  @summary("Delete phase for plan")
  delete(@path planId: ULID, @path planPhaseKey: Key): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}
