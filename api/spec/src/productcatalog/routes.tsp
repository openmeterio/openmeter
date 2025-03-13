import "../rest.tsp";

namespace OpenMeter.ProductCatalog;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

alias PlanIdOrKey = ULID | Key;

@route("/api/v1/plans")
@tag("Product Catalog (Beta)")
interface Plans {
  /**
   * List all plans.
   */
  @list
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
    keyVersion?: Record<integer[]>,

    /**
     * Filter by plan.currency attribute
     */
    @query(#{ explode: true })
    currency?: CurrencyCode[],

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
    @body body: Plan;
  } | OpenMeter.CommonErrors;

  /**
   * Update plan by id.
   */
  @put
  @operationId("updatePlan")
  @summary("Update a plan")
  update(
    @path planId: ULID,
    @body request: TypeSpec.Rest.Resource.ResourceReplaceModel<Plan>,
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
  delete(@path planId: ULID): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Publish a plan version.
   */
  @post
  @route("/{planId}/publish")
  @operationId("publishPlan")
  @summary("Publish plan")
  publish(
    @path planId: ULID,
  ): Plan | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Archive a plan version.
   */
  @post
  @route("/{planId}/archive")
  @operationId("archivePlan")
  @summary("Archive plan version")
  archive(
    @path planId: ULID,
  ): Plan | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Create a new draft version from plan.
   * It returns error if there is already a plan in draft or planId does not reference the latest published version.
   */
  @post
  @route("/{planIdOrKey}/next")
  @operationId("nextPlan")
  @summary("New draft plan")
  next(
    @extension("x-go-type", "string")
    @path
    planIdOrKey: PlanIdOrKey,
  ): {
    @statusCode _: 201;
    @body body: Plan;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

@route("/api/v1/subscriptions")
@tag("Subscriptions (Beta)")
interface Subscriptions {
  @get
  @sharedRoute
  @operationId("getSubscription")
  @summary("Get subscription")
  getExpanded(
    @path subscriptionId: ULID,

    /**
     * The time at which the subscription should be queried. If not provided the current time is used.
     */
    @query(#{ explode: true }) at?: DateTime,
  ): SubscriptionExpanded | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  @post
  @operationId("createSubscription")
  @summary("Create subscription")
  create(@body body: SubscriptionCreate): {
    @statusCode _: 201;
    @body body: Subscription;
  } | OpenMeter.ConflictError | OpenMeter.CommonErrors;

  /**
   * Batch processing commands for manipulating running subscriptions.
   * The key format is `/phases/{phaseKey}` or `/phases/{phaseKey}/items/{itemKey}`.
   */
  @patch
  @operationId("editSubscription")
  @summary("Edit subscription")
  @route("/{subscriptionId}")
  @parameterVisibility("create")
  edit(
    @path subscriptionId: ULID,

    @body
    body: SubscriptionEdit,
  ):
    | Subscription
    | OpenMeter.ConflictError
    | OpenMeter.NotFoundError
    | OpenMeter.CommonErrors;

  /**
   * Closes a running subscription and starts a new one according to the specification.
   * Can be used for upgrades, downgrades, and plan changes.
   */
  @post
  @operationId("changeSubscription")
  @summary("Change subscription")
  @route("/{subscriptionId}/change")
  change(
    @path subscriptionId: ULID,

    @body
    body: SubscriptionChange,
  ):
    | SubscriptionChangeResponseBody
    | OpenMeter.ConflictError
    | OpenMeter.NotFoundError
    | OpenMeter.CommonErrors;

  /**
   * Migrates the subscripiton to the provided version of the current plan.
   * If possible, the migration will be done immediately.
   * If not, the migration will be scheduled to the end of the current billing period.
   */
  @post
  @operationId("migrateSubscription")
  @summary("Migrate subscription")
  @route("/{subscriptionId}/migrate")
  migrate(
    @path subscriptionId: ULID,
    @body body: {
      @minValue(1)
      targetVersion?: integer;
    },
  ):
    | SubscriptionChangeResponseBody
    | OpenMeter.ConflictError
    | OpenMeter.NotFoundError
    | OpenMeter.CommonErrors;

  /**
   * Restores a canceled subscription.
   * Any subscription scheduled to start later will be deleted and this subscription will be continued indefinitely.
   */
  @post
  @operationId("restoreSubscription")
  @summary("Restore subscription")
  @route("/{subscriptionId}/restore")
  restore(
    @path subscriptionId: ULID,
  ): Subscription | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Cancels the subscription.
   * Will result in a scheduling conflict if there are other subscriptions scheduled to start after the cancellation time.
   */
  @post
  @operationId("cancelSubscription")
  @summary("Cancel subscription")
  @route("/{subscriptionId}/cancel")
  cancel(
    @path subscriptionId: ULID,
    @body body: {
      /**
       * If not provided the subscription is canceled immediately.
       */
      timing?: SubscriptionTiming;
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
  unscheduleCancelation(@path subscriptionId: ULID):
    | Subscription
    | OpenMeter.ConflictError
    | OpenMeter.NotFoundError
    | OpenMeter.CommonErrors;
}
