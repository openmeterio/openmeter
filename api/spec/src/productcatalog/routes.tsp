import "../rest.tsp";
import "./subscriptionaddon.tsp";
import "./errors.tsp";

namespace OpenMeter;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@route("/api/v1/features")
@tag("Product Catalog")
@friendlyName("Features")
interface FeaturesEndpoints {
  /**
   * List features.
   */
  @get
  @operationId("listFeatures")
  @summary("List features")
  list(
    /**
     * Filter by meterSlug
     */
    @query(#{ explode: true })
    meterSlug?: string[],

    /**
     * Include archived features in response.
     */
    @query
    includeArchived?: boolean = false,

    ...QueryPagination,
    ...QueryLimitOffset,
    ...QueryOrdering<FeatureOrderBy>,
  ): {
    @statusCode _: 200;
    @body body: ListFeaturesResult;
  } | CommonErrors;

  /**
   * Features are either metered or static. A feature is metered if meterSlug is provided at creation.
   * For metered features you can pass additional filters that will be applied when calculating feature usage, based on the meter's groupBy fields.
   * Meters with SUM, COUNT, UNIQUE_COUNT and LATEST aggregations are supported for features.
   * Features cannot be updated later, only archived.
   */
  @post
  @operationId("createFeature")
  @summary("Create feature")
  create(@body feature: FeatureCreateInputs): {
    @statusCode _: 201;
    @body body: Feature;
  } | CommonErrors;

  /**
   * Get a feature by ID.
   */
  @get
  @operationId("getFeature")
  @summary("Get feature")
  get(@path featureId: string): Feature | NotFoundError | CommonErrors;

  /**
   * Archive a feature by ID.
   *
   * Once a feature is archived it cannot be unarchived. If a feature is archived, new entitlements cannot be created for it, but archiving the feature does not affect existing entitlements.
   * This means, if you want to create a new feature with the same key, and then create entitlements for it, the previous entitlements have to be deleted first on a per subject basis.
   */
  @delete
  @operationId("deleteFeature")
  @summary("Delete feature")
  delete(@path featureId: string): {
    @statusCode _: 204;
  } | NotFoundError | CommonErrors;

  /**
   * Query the cost of a feature based on its unit cost configuration and underlying meter usage.
   * For features with manual unit cost, the cost is usage × fixed amount.
   * For features with LLM unit cost, the cost is resolved per row from the LLM cost database.
   * The feature must have a meter and unit cost configured.
   */
  @get
  @route("/{featureId}/cost")
  @operationId("queryFeatureCost")
  @summary("Query feature cost")
  queryFeatureCost(
    @path featureId: string,
    ...MeterQuery,
  ): FeatureCostQueryResult | NotFoundError | CommonErrors;
}

@route("/api/v1/plans")
@tag("Product Catalog")
@friendlyName("Plans")
interface PlansEndpoints {
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

    ...QueryPagination,
    ...QueryOrdering<PlanOrderBy>,
  ): PaginatedResponse<Plan> | CommonErrors;

  /**
   * Create a new plan.
   */
  @post
  @operationId("createPlan")
  @summary("Create a plan")
  create(@body request: TypeSpec.Rest.Resource.ResourceCreateModel<Plan>): {
    @statusCode _: 201;
    @body body: Plan;
  } | CommonErrors;

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
  ): Plan | NotFoundError | CommonErrors;

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
  ): Plan | NotFoundError | CommonErrors;

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
  } | NotFoundError | CommonErrors;

  /**
   * Publish a plan version.
   */
  @post
  @route("/{planId}/publish")
  @operationId("publishPlan")
  @summary("Publish plan")
  publish(@path planId: ULID): Plan | NotFoundError | CommonErrors;

  /**
   * Archive a plan version.
   */
  @post
  @route("/{planId}/archive")
  @operationId("archivePlan")
  @summary("Archive plan version")
  archive(@path planId: ULID): Plan | NotFoundError | CommonErrors;

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
  } | NotFoundError | CommonErrors;
}

@route("/api/v1/plans/{planId}/addons")
@tag("Product Catalog")
@friendlyName("PlanAddons")
interface PlanAddonsEndpoints {
  /**
   * List all available add-ons for plan.
   */
  @get
  @operationId("listPlanAddons")
  @summary("List all available add-ons for plan")
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

    ...QueryPagination,
    ...QueryOrdering<PlanAddonOrderBy>,
  ): PaginatedResponse<PlanAddon> | NotFoundError | CommonErrors;

  /**
   * Create new add-on assignment for plan.
   */
  @post
  @operationId("createPlanAddon")
  @summary("Create new add-on assignment for plan")
  create(@path planId: ULID, @body body: PlanAddonCreate):
    | {
        @statusCode _: 201;
        @body body: PlanAddon;
      }
    | ConflictError
    | NotFoundError
    | CommonErrors;

  /**
   * Update add-on assignment for plan.
   */
  @put
  @route("/{planAddonId}")
  @operationId("updatePlanAddon")
  @summary("Update add-on assignment for plan")
  update(
    @path planId: ULID,
    @path planAddonId: ULID,
    @body body: TypeSpec.Rest.Resource.ResourceReplaceModel<PlanAddon>,
  ): PlanAddon | NotFoundError | CommonErrors;

  /**
   * Get add-on assignment for plan by id.
   */
  @get
  @route("/{planAddonId}")
  @operationId("getPlanAddon")
  @summary("Get add-on assignment for plan")
  get(
    @path planId: ULIDOrKey,
    @path planAddonId: ULIDOrKey,
  ): PlanAddon | NotFoundError | CommonErrors;

  /**
   * Delete add-on assignment for plan.
   *
   * Once a plan is deleted it cannot be undeleted.
   */
  @delete
  @route("/{planAddonId}")
  @operationId("deletePlanAddon")
  @summary("Delete add-on assignment for plan")
  delete(@path planId: ULID, @path planAddonId: ULID): {
    @statusCode _: 204;
  } | NotFoundError | CommonErrors;
}

@route("/api/v1/addons")
@tag("Product Catalog")
@friendlyName("Addons")
interface AddonsEndpoints {
  /**
   * List all add-ons.
   */
  @list
  @operationId("listAddons")
  @summary("List add-ons")
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

    ...QueryPagination,
    ...QueryOrdering<AddonOrderBy>,
  ): PaginatedResponse<Addon> | CommonErrors;

  /**
   * Create a new add-on.
   */
  @post
  @operationId("createAddon")
  @summary("Create an add-on")
  create(@body request: TypeSpec.Rest.Resource.ResourceCreateModel<Addon>): {
    @statusCode _: 201;
    @body body: Addon;
  } | CommonErrors;

  /**
   * Update add-on by id.
   */
  @put
  @route("/{addonId}")
  @operationId("updateAddon")
  @summary("Update add-on")
  update(
    @path addonId: ULID,
    @body request: TypeSpec.Rest.Resource.ResourceReplaceModel<Addon>,
  ): Addon | NotFoundError | CommonErrors;

  /**
   * Get add-on by id or key. The latest published version is returned if latter is used.
   */
  @get
  @route("/{addonId}")
  @operationId("getAddon")
  @summary("Get add-on")
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
  ): Addon | NotFoundError | CommonErrors;

  /**
   * Soft delete add-on by id.
   *
   * Once a add-on is deleted it cannot be undeleted.
   */
  @delete
  @route("/{addonId}")
  @operationId("deleteAddon")
  @summary("Delete add-on")
  delete(@path addonId: ULID): {
    @statusCode _: 204;
  } | NotFoundError | CommonErrors;

  /**
   * Publish a add-on version.
   */
  @post
  @route("/{addonId}/publish")
  @operationId("publishAddon")
  @summary("Publish add-on")
  publish(@path addonId: ULID): Addon | NotFoundError | CommonErrors;

  /**
   * Archive a add-on version.
   */
  @post
  @route("/{addonId}/archive")
  @operationId("archiveAddon")
  @summary("Archive add-on version")
  archive(@path addonId: ULID): Addon | NotFoundError | CommonErrors;
}

@route("/api/v1/subscriptions")
@tag("Subscriptions")
@friendlyName("Subscriptions")
interface SubscriptionsEndpoints {
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
  ): SubscriptionExpanded | NotFoundError | CommonErrors;

  @post
  @operationId("createSubscription")
  @summary("Create subscription")
  create(@body body: SubscriptionCreate): {
    @statusCode _: 201;
    @body body: Subscription;
  } | CommonSubscriptionErrors;

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
  ): Subscription | NotFoundError | CommonSubscriptionErrors;

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
  ): SubscriptionChangeResponseBody | NotFoundError | CommonSubscriptionErrors;

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
      /**
       * Timing configuration for the migration, when the migration should take effect.
       * If not supported by the subscription, 400 will be returned.
       */
      timing?: SubscriptionTiming = SubscriptionTimingEnum.Immediate;

      /**
       * The version of the plan to migrate to.
       * If not provided, the subscription will migrate to the latest version of the current plan.
       */
      @minValue(1)
      targetVersion?: integer;

      /**
       * The key of the phase to start the subscription in.
       * If not provided, the subscription will start in the first phase of the plan.
       */
      @minLength(1)
      startingPhase?: string;

      /**
       * The billing anchor of the subscription. The provided date will be normalized according to the billing cadence to the nearest recurrence before start time. If not provided, the previous subscription billing anchor will be used.
       */
      billingAnchor?: DateTime;
    },
  ): SubscriptionChangeResponseBody | NotFoundError | CommonSubscriptionErrors;

  /**
   * Restores a canceled subscription.
   * Any subscription scheduled to start later will be deleted and this subscription will be continued indefinitely.
   */
  #deprecated "API will be removed by 2025-10-06"
  @post
  @operationId("restoreSubscription")
  @summary("Restore subscription")
  @route("/{subscriptionId}/restore")
  restore(
    @path subscriptionId: ULID,
  ): Subscription | NotFoundError | CommonErrors;

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
  ): Subscription | NotFoundError | CommonSubscriptionErrors;

  /**
   * Cancels the scheduled cancelation.
   */
  @post
  @operationId("unscheduleCancelation")
  @summary("Unschedule cancelation")
  @route("/{subscriptionId}/unschedule-cancelation")
  unscheduleCancelation(
    @path subscriptionId: ULID,
  ): Subscription | NotFoundError | CommonSubscriptionErrors;

  /**
   * Deletes a subscription. Only scheduled subscriptions can be deleted.
   */
  @delete
  @operationId("deleteSubscription")
  @summary("Delete subscription")
  @route("/{subscriptionId}")
  delete(@path subscriptionId: ULID): {
    @statusCode _: 204;
  } | NotFoundError | CommonSubscriptionErrors;
}

@route("/api/v1/subscriptions/{subscriptionId}/addons")
@tag("Subscriptions")
@friendlyName("SubscriptionAddons")
interface SubscriptionAddonsEndpoints {
  /**
   * Create a new subscription addon, either providing the key or the id of the addon.
   */
  @post
  @operationId("createSubscriptionAddon")
  @summary("Create subscription addon")
  create(
    @path subscriptionId: ULID,

    @body
    request: SubscriptionAddonCreate,
  ):
    | {
        @statusCode _: 201;
        @body body: SubscriptionAddon;
      }
    | ConflictError
    | NotFoundError
    | CommonErrors;

  /**
   * List all addons of a subscription. In the returned list will match to a set unique by addonId.
   */
  @get
  @operationId("listSubscriptionAddons")
  @summary("List subscription addons")
  list(
    @path subscriptionId: ULID, // Should we implement pagination? This should never be a large list.
  ): SubscriptionAddon[] | NotFoundError | CommonErrors;

  /**
   * Get a subscription addon by id.
   */
  @get
  @route("/{subscriptionAddonId}")
  @operationId("getSubscriptionAddon")
  @summary("Get subscription addon")
  get(
    @path subscriptionId: ULID,
    @path subscriptionAddonId: ULID,
  ): SubscriptionAddon | NotFoundError | CommonErrors;

  /**
   * Updates a subscription addon (allows changing the quantity: purchasing more instances or cancelling the current instances)
   */
  @patch
  @route("/{subscriptionAddonId}")
  @operationId("updateSubscriptionAddon")
  @summary("Update subscription addon")
  update(
    @path subscriptionId: ULID,
    @path subscriptionAddonId: ULID,

    @body
    body: TypeSpec.Rest.Resource.ResourceCreateOrUpdateModel<SubscriptionAddon>,
  ): SubscriptionAddon | NotFoundError | CommonErrors;
}
