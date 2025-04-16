import "../rest.tsp";
import "./subscriptionaddon.tsp";

namespace OpenMeter.ProductCatalog;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@route("/api/v1/plans")
@tag("Product Catalog")
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
     * Only return plans with the given status.
     *
     * Usage:
     * - `?status=active`: return only the currently active plan
     * - `?status=draft`: return only the draft plan
     * - `?status=archived`: return only the archived plans
     */
    @query(#{ explode: true })
    status?: PlanStatus[],

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
  @route("/{planId}")
  @operationId("updatePlan")
  @summary("Update a plan")
  update(
    @path planId: ULID,
    @body body: TypeSpec.Rest.Resource.ResourceReplaceModel<Plan>,
  ): Plan | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Get a plan by id or key. The latest published version is returned if latter is used.
   */
  @get
  @route("/{planId}")
  @operationId("getPlan")
  @summary("Get plan")
  get(
    @extension("x-go-type", "string")
    @path
    planId: ULIDOrKey,

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
  @route("/{planId}")
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
  #deprecated "Use createPlan instead"
  @post
  @route("/{planIdOrKey}/next")
  @operationId("nextPlan")
  @summary("New draft plan")
  next(
    @extension("x-go-type", "string")
    @path
    planIdOrKey: ULIDOrKey,
  ): {
    @statusCode _: 201;
    @body body: Plan;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

@route("/api/v1/plans/{planId}/addons")
@tag("Product Catalog")
interface PlanAddons {
  /**
   * List all available add-ons for plan.
   */
  @get
  @operationId("listPlanAddons")
  @summary("List all available add-ons for plan")
  @extension("x-internal", true)
  list(
    @path planId: ULIDOrKey,

    /**
     * Include deleted plan add-on assignments.
     *
     * Usage: `?includeDeleted=true`
     */
    @query
    @example(true)
    includeDeleted?: boolean = false,

    /**
     * Filter by addon.id attribute.
     */
    @query(#{ explode: true })
    id?: ULID[],

    /**
     * Filter by addon.key attribute.
     */
    @query(#{ explode: true })
    key?: Key[],

    /**
     * Filter by addon.key and addon.version attributes.
     */
    @query(#{ explode: true, style: "deepObject" })
    keyVersion?: Record<integer[]>,

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryOrdering<PlanAddonOrderBy>,
  ): OpenMeter.PaginatedResponse<PlanAddon> | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Create new add-on assignment for plan.
   */
  @post
  @operationId("createPlanAddon")
  @summary("Create new add-on assignment for plan")
  @extension("x-internal", true)
  create(@path planId: ULID, @body body: PlanAddonCreate):
    | {
        @statusCode _: 201;
        @body body: PlanAddon;
      }
    | OpenMeter.ConflictError
    | OpenMeter.NotFoundError
    | OpenMeter.CommonErrors;

  /**
   * Update add-on assignment for plan.
   */
  @put
  @route("/{planAddonId}")
  @operationId("updatePlanAddon")
  @summary("Update add-on assignment for plan")
  @extension("x-internal", true)
  update(
    @path planId: ULID,
    @path planAddonId: ULID,
    @body body: TypeSpec.Rest.Resource.ResourceReplaceModel<PlanAddon>,
  ): PlanAddon | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Get add-on assignment for plan by id.
   */
  @get
  @route("/{planAddonId}")
  @operationId("getPlanAddon")
  @summary("Get add-on assignment for plan")
  @extension("x-internal", true)
  get(
    @path planId: ULIDOrKey,
    @path planAddonId: ULIDOrKey,
  ): PlanAddon | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Delete add-on assignment for plan.
   *
   * Once a plan is deleted it cannot be undeleted.
   */
  @delete
  @route("/{planAddonId}")
  @operationId("deletePlanAddon")
  @summary("Delete add-on assignment for plan")
  @extension("x-internal", true)
  delete(@path planId: ULID, @path planAddonId: ULID): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

@route("/api/v1/addons")
@tag("Product Catalog")
interface Addons {
  /**
   * List all add-ons.
   */
  @list
  @operationId("listAddons")
  @summary("List add-ons")
  @extension("x-internal", true)
  list(
    /**
     * Include deleted add-ons in response.
     *
     * Usage: `?includeDeleted=true`
     */
    @query
    @example(true)
    includeDeleted?: boolean = false,

    /**
     * Filter by addon.id attribute
     */
    @query(#{ explode: true })
    id?: ULID[],

    /**
     * Filter by addon.key attribute
     */
    @query(#{ explode: true })
    key?: Key[],

    /**
     * Filter by addon.key and addon.version attributes
     */
    @query(#{ explode: true, style: "deepObject" })
    keyVersion?: Record<integer[]>,

    /**
     * Only return add-ons with the given status.
     *
     * Usage:
     * - `?status=active`: return only the currently active add-ons
     * - `?status=draft`: return only the draft add-ons
     * - `?status=archived`: return only the archived add-ons
     */
    @query(#{ explode: true })
    status?: AddonStatus[],

    /**
     * Filter by addon.currency attribute
     */
    @query(#{ explode: true })
    currency?: CurrencyCode[],

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryOrdering<AddonOrderBy>,
  ): OpenMeter.PaginatedResponse<Addon> | OpenMeter.CommonErrors;

  /**
   * Create a new add-on.
   */
  @post
  @operationId("createAddon")
  @summary("Create an add-on")
  @extension("x-internal", true)
  create(@body request: TypeSpec.Rest.Resource.ResourceCreateModel<Addon>): {
    @statusCode _: 201;
    @body body: Addon;
  } | OpenMeter.CommonErrors;

  /**
   * Update add-on by id.
   */
  @put
  @route("/{addonId}")
  @operationId("updateAddon")
  @summary("Update add-on")
  @extension("x-internal", true)
  update(
    @path addonId: ULID,
    @body request: TypeSpec.Rest.Resource.ResourceReplaceModel<Addon>,
  ): Addon | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Get add-on by id or key. The latest published version is returned if latter is used.
   */
  @get
  @route("/{addonId}")
  @operationId("getAddon")
  @summary("Get add-on")
  @extension("x-internal", true)
  get(
    @extension("x-go-type", "string")
    @path
    addonId: ULIDOrKey,

    /**
     * Include latest version of the add-on instead of the version in active state.
     *
     * Usage: `?includeLatest=true`
     */
    @query
    @example(true)
    includeLatest?: boolean = false,
  ): Addon | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Soft delete add-on by id.
   *
   * Once a add-on is deleted it cannot be undeleted.
   */
  @delete
  @route("/{addonId}")
  @operationId("deleteAddon")
  @summary("Delete add-on")
  @extension("x-internal", true)
  delete(@path addonId: ULID): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Publish a add-on version.
   */
  @post
  @route("/{addonId}/publish")
  @operationId("publishAddon")
  @summary("Publish add-on")
  @extension("x-internal", true)
  publish(
    @path addonId: ULID,
  ): Addon | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Archive a add-on version.
   */
  @post
  @route("/{addonId}/archive")
  @operationId("archiveAddon")
  @summary("Archive add-on version")
  @extension("x-internal", true)
  archive(
    @path addonId: ULID,
  ): Addon | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

@route("/api/v1/subscriptions")
@tag("Subscriptions")
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
  @parameterVisibility(Lifecycle.Create)
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

  /**
   * Deletes a subscription. Only scheduled subscriptions can be deleted.
   */
  @delete
  @operationId("deleteSubscription")
  @summary("Delete subscription")
  @route("/{subscriptionId}")
  delete(@path subscriptionId: ULID): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

@route("/api/v1/subscriptions/{subscriptionId}/addons")
@tag("Subscriptions")
interface SubscriptionAddons {
  /**
   * Create a new subscription addon, either providing the key or the id of the addon.
   */
  @post
  @operationId("createSubscriptionAddon")
  @summary("Create subscription addon")
  @extension("x-internal", true)
  create(
    @path subscriptionId: ULID,

    @body
    request: SubscriptionAddonCreate,
  ):
    | {
        @statusCode _: 201;
        @body body: SubscriptionAddon;
      }
    | OpenMeter.ConflictError
    | OpenMeter.NotFoundError
    | OpenMeter.CommonErrors;

  /**
   * List all addons of a subscription. In the returned list will match to a set unique by addonId.
   */
  @get
  @operationId("listSubscriptionAddons")
  @summary("List subscription addons")
  @extension("x-internal", true)
  list(
    @path subscriptionId: ULID, // Should we implement pagination? This should never be a large list.
  ): SubscriptionAddon[] | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Get a subscription addon by id.
   */
  @get
  @route("/{subscriptionAddonId}")
  @operationId("getSubscriptionAddon")
  @summary("Get subscription addon")
  @extension("x-internal", true)
  get(
    @path subscriptionId: ULID,
    @path subscriptionAddonId: ULID,
  ): SubscriptionAddon | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Updates a subscription addon (allows changing the quantity: purchasing more instances or cancelling the current instances)
   */
  @patch
  @route("/{subscriptionAddonId}")
  @operationId("updateSubscriptionAddon")
  @summary("Update subscription addon")
  @extension("x-internal", true)
  update(
    @path subscriptionId: ULID,
    @path subscriptionAddonId: ULID,

    @body
    body: TypeSpec.Rest.Resource.ResourceCreateOrUpdateModel<SubscriptionAddon>,
  ): SubscriptionAddon | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}
