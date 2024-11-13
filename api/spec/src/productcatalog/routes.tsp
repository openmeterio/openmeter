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
  @extension("x-internal", true)
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
    keyVersion?: Record<integer[]>,

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryOrdering<PlanOrderBy>,
  ): OpenMeter.PaginatedResponse<Plan> | OpenMeter.CommonErrors;

  /**
   * Create a new plan.
   */
  @post
  @operationId("createPlan")
  @summary("Create a plan")
  @extension("x-internal", true)
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
  @extension("x-internal", true)
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
  @extension("x-internal", true)
  get(
    @extension("x-go-type", "string")
    @path
    planId: PlanIdOrKey,

    /**
     * Include latest version of the Plan instead of the version in active state.
     *
     * Usage: `?includeLatest=true`
     */
    @query
    @example(true)
    includeLatest?: boolean = false,
  ): Plan | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Soft delete plan by plan.id.
   *
   * Once a plan is deleted it cannot be undeleted.
   */
  @delete
  @operationId("deletePlan")
  @summary("Delete plan")
  @extension("x-internal", true)
  delete(@path planId: ULID): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Publish a plan version.
   */
  @post
  @route("/{planId}/publish")
  @sharedRoute
  @operationId("publishPlan")
  @summary("Publish plan")
  @extension("x-internal", true)
  publish(
    @path planId: ULID,
  ): Plan | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Archive a plan version.
   */
  @post
  @route("/{planId}/archive")
  @sharedRoute
  @operationId("archivePlan")
  @summary("Archive plan version")
  @extension("x-internal", true)
  archive(
    @path planId: ULID,
  ): Plan | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Create a new draft version from plan.
   * It returns error if there is already a plan in draft or planId does not reference the latest published version.
   */
  @post
  @route("/{planIdOrKey}/next")
  @sharedRoute
  @operationId("nextPlan")
  @summary("New draft plan")
  @extension("x-internal", true)
  next(
    @extension("x-go-type", "string")
    @path
    planIdOrKey: PlanIdOrKey,
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
  @extension("x-internal", true)
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
  @operationId("createPlanPhase")
  @summary("Create new phase in plan")
  @extension("x-internal", true)
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
  @operationId("updatePlanPhase")
  @summary("Update phase in plan")
  @extension("x-internal", true)
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
  @operationId("getPlanPhase")
  @summary("Get phase for plan")
  @extension("x-internal", true)
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
  @operationId("deletePlanPhase")
  @summary("Delete phase for plan")
  @extension("x-internal", true)
  delete(@path planId: ULID, @path planPhaseKey: Key): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

@route("/api/v1/subscriptions")
@tag("Subscriptions")
interface Subscriptions {
  @get
  @sharedRoute
  @operationId("getSubscription")
  @summary("Get subscription")
  @extension("x-internal", true)
  get(
    @path subscriptionId: ULID,
    @query expand?: false,

    /**
     * The time at which the subscription should be queried. If not provided the current time is used.
     */
    @query at?: DateTime,
  ): Subscription | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  @get
  @sharedRoute
  @operationId("getSubscription")
  @summary("Get subscription")
  @extension("x-internal", true)
  getExpanded(
    @path subscriptionId: ULID,
    @query expand: true,

    /**
     * The time at which the subscription should be queried. If not provided the current time is used.
     */
    @query at?: DateTime,
  ): SubscriptionExpanded | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  @post
  @operationId("createSubscription")
  @summary("Create subscription")
  @extension("x-internal", true)
  create(
    @body body: CreateSubscriptionRequestBody,
  ): Subscription | OpenMeter.ConflictError | OpenMeter.CommonErrors;

  /**
   * Batch processing commands for manipulating running subscriptions.
   * The key format is `/phases/{phaseKey}` or `/phases/{phaseKey}/items/{itemKey}`.
   *
   * Add operations insert a new member based on the creation input without altering the existing members.
   *
   * Remove operations remove the member from the collection / document.
   *
   * The extend operation extends the specific phase if possible, while delaying all subsequent phases by the same amount.
   */
  @patch
  @operationId("editSubscription")
  @summary("Edit subscription")
  @route("/{subscriptionId}")
  @extension("x-internal", true)
  edit(
    @path subscriptionId: ULID,

    @body
    @maxItems(100)
    body: SubscriptionPatch[],
  ):
    | Subscription
    | OpenMeter.ConflictError
    | OpenMeter.NotFoundError
    | OpenMeter.CommonErrors;

  /**
   * Migrates the subscripiton to the procided version of the plan.
   */
  @post
  @operationId("migrateSubscription")
  @summary("Migrate subscription")
  @route("/{subscriptionId}/migrate")
  @extension("x-internal", true)
  migrate(
    @path subscriptionId: ULID,
    @body body: {
      @minValue(1)
      targetVersion: integer;
    },
  ):
    | Subscription
    | OpenMeter.ConflictError
    | OpenMeter.NotFoundError
    | OpenMeter.CommonErrors;

  /**
   * Cancels the subscription.
   */
  @post
  @operationId("cancelSubscription")
  @summary("Cancel subscription")
  @route("/{subscriptionId}/cancel")
  @extension("x-internal", true)
  cancel(
    @path subscriptionId: ULID,
    @body body: {
      /**
       * If not provided the subscription is canceled immediately.
       */
      effectiveDate?: DateTime;
    },
  ):
    | Subscription
    | OpenMeter.ConflictError
    | OpenMeter.NotFoundError
    | OpenMeter.CommonErrors;

  /**
   * Cancels the scheduled cancelation.
   */
  @post
  @operationId("unscheduleCancelation")
  @summary("Unschedule cancelation")
  @route("/{subscriptionId}/unschedule-cancelation")
  @extension("x-internal", true)
  unscheduleCancelation(@path subscriptionId: ULID):
    | Subscription
    | OpenMeter.ConflictError
    | OpenMeter.NotFoundError
    | OpenMeter.CommonErrors;
}
