import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../shared/index.tsp";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "./subscription.tsp";
import "./subscriptionaddon.tsp";
import "./edit.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Subscriptions;

/**
 * Request for canceling a subscription.
 */
@friendlyName("BillingSubscriptionCancel")
model SubscriptionCancel {
  /**
   * If not provided the subscription is canceled immediately.
   */
  timing?: SubscriptionEditTiming = SubscriptionEditTimingEnum.Immediate;
}

/**
 * Request for changing a subscription.
 */
@friendlyName("BillingSubscriptionChange")
model SubscriptionChange {
  // `timing` is omitted from the spread and redeclared as required below: on
  // create it is an optional scheduling hint, but a change must always say when it
  // takes effect.
  ...OmitProperties<
    SubscriptionCreate,
    "customer_id" | "customer_key" | "timing"
  >;

  /**
   * Timing configuration for the change, when the change should take effect. For
   * changing a subscription, the accepted values depend on the subscription
   * configuration.
   */
  timing: SubscriptionEditTiming;
}

/**
 * Response for changing a subscription.
 */
@friendlyName("BillingSubscriptionChangeResponse")
model SubscriptionChangeResponse {
  /**
   * The current subscription before the change.
   */
  current: Subscription;

  /**
   * The new state of the subscription after the change.
   */
  next: Subscription;
}

/**
 * Request for migrating to a later version of the subscription's current plan.
 */
@friendlyName("BillingSubscriptionMigrate")
model SubscriptionMigrate {
  /**
   * When the migration takes effect: immediately (default), at the next billing
   * cycle, or at an explicit billing-aligned timestamp. In-place migrations may
   * target a later phase when the phase timelines match. The target plan reference
   * is saved now; changed items take effect at the requested time.
   */
  timing?: SubscriptionEditTiming = SubscriptionEditTimingEnum.Immediate;

  /**
   * A strictly later version of the current plan. Omit to use its latest version.
   */
  @minValue(1)
  target_version?: integer;

  /**
   * Explicitly replace the subscription, starting in this target plan phase. Always
   * selects replacement, even when the phase matches the current one. Replacement
   * may produce billing adjustments and does not transfer addons.
   */
  @minLength(1)
  starting_phase?: string;

  /**
   * Providing a different billing anchor explicitly replaces the subscription. The
   * supplied anchor is preserved and may be before or after the replacement's start
   * time. Omit to retain the existing anchor.
   */
  billing_anchor?: Shared.DateTime;
}

/**
 * Response for migrating a subscription.
 */
@friendlyName("BillingSubscriptionMigrateResponse")
model SubscriptionMigrateResponse {
  /**
   * The original subscription's own fields returned by the migration. For an
   * in-place migration this is the before snapshot; for replacement it includes the
   * cancellation. Does not include phases or the current billing period.
   */
  current: SubscriptionBase;

  /**
   * The resulting subscription, including its phases and items.
   */
  next: Subscription;
}

/**
 * Filter options for listing subscriptions.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "plan_id and plan_key should not be grouped"
@friendlyName("ListSubscriptionsParamsFilter")
model ListSubscriptionsParamsFilter {
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  id?: Common.ULIDFieldFilter;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  customer_id?: Common.ULIDFieldFilter;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  status?: Common.StringFieldFilterExact;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  plan_id?: Common.ULIDFieldFilter;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  plan_key?: Common.StringFieldFilterExact;
}

interface SubscriptionsOperations {
  @post
  @operationId("create-subscription")
  @summary("Create subscription")
  @extension(Shared.UnstableExtension, true)
  create(@body subscription: SubscriptionCreate):
    | Shared.CreateResponse<Subscription>
    | Common.NotFound
    | Common.Conflict
    | Common.ErrorResponses;

  @get
  @operationId("list-subscriptions")
  @summary("List subscriptions")
  @extension(Shared.UnstableExtension, true)
  list(
    ...Common.PagePaginationQuery,

    /**
     * Sort subscriptions returned in the response. Supported sort attributes are:
     *
     * - `id`
     * - `active_from` (default)
     * - `active_to`
     *
     * The `asc` suffix is optional as the default sort order is ascending. The `desc`
     * suffix is used to specify a descending order.
     */
    @query(#{ name: "sort" })
    sort?: Common.SortQuery,

    /**
     * Filter subscriptions.
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListSubscriptionsParamsFilter,
  ):
    | Shared.PagePaginatedResponse<Subscription>
    | Common.NotFound
    | Common.ErrorResponses;

  @get
  @operationId("get-subscription")
  @summary("Get subscription")
  @extension(Shared.UnstableExtension, true)
  get(@path subscriptionId: Shared.ULID):
    | Shared.GetResponse<Subscription>
    | Common.NotFound
    | Common.ErrorResponses;

  /**
   * Cancels the subscription. Will result in a scheduling conflict if there are
   * other subscriptions scheduled to start after the cancelation time.
   */
  @post
  @operationId("cancel-subscription")
  @summary("Cancel subscription")
  @extension(Shared.UnstableExtension, true)
  @route("/{subscriptionId}/cancel")
  cancel(@path subscriptionId: Shared.ULID, @body body: SubscriptionCancel):
    | Shared.UpdateResponse<Subscription>
    | Common.NotFound
    | Common.Conflict
    | Common.ErrorResponses;

  /**
   * Unschedules the subscription cancelation.
   */
  @post
  @operationId("unschedule-cancelation")
  @summary("Unschedule subscription cancelation")
  @extension(Shared.UnstableExtension, true)
  @route("/{subscriptionId}/unschedule-cancelation")
  unscheduleCancelation(@path subscriptionId: Shared.ULID):
    | Shared.UpdateResponse<Subscription>
    | Common.NotFound
    | Common.Conflict
    | Common.ErrorResponses;

  /**
   * Deletes a scheduled subscription that has not yet become active, removing it and
   * resolving any scheduling conflict it was holding. This is distinct from
   * canceling: cancel ends a running subscription, whereas unscheduling removes a
   * not-yet-active one. Only scheduled subscriptions can be unscheduled;
   * unscheduling an active or already-started subscription is rejected.
   */
  @post
  @operationId("unschedule-subscription")
  @summary("Unschedule subscription")
  @extension(Shared.UnstableExtension, true)
  @route("/{subscriptionId}/unschedule")
  unschedule(@path subscriptionId: Shared.ULID):
    | Shared.DeleteResponse
    | Common.NotFound
    | Common.ErrorResponses;

  /**
   * Restores the subscription by deleting any later-scheduled successor
   * subscriptions and continuing this one indefinitely. This is the inverse of a
   * future-dated change, which schedules a successor. Restore is not available when
   * multi-subscription is enabled.
   */
  @post
  @operationId("restore-subscription")
  @summary("Restore subscription")
  @extension(Shared.UnstableExtension, true)
  @route("/{subscriptionId}/restore")
  restore(@path subscriptionId: Shared.ULID):
    | Shared.UpdateResponse<Subscription>
    | Common.NotFound
    | Common.Conflict
    | Common.ErrorResponses;

  /**
   * Closes a running subscription and starts a new one according to the
   * specification. Can be used for upgrades, downgrades, and plan changes.
   */
  @post
  @operationId("change-subscription")
  @summary("Change subscription")
  @extension(Shared.UnstableExtension, true)
  @route("/{subscriptionId}/change")
  change(
    @path subscriptionId: Shared.ULID,

    @body
    body: SubscriptionChange,
  ):
    | SubscriptionChangeResponse
    | Common.NotFound
    | Common.Conflict
    | Common.ErrorResponses;

  /**
   * Migrates to a later version of the current plan. With starting_phase omitted and
   * billing_anchor omitted or unchanged, migration amends the subscription in place:
   * unchanged items retain their service periods and both response entries have the
   * same ID. Existing addons must remain compatible with the target plan.
   * Incompatible phase timelines or billing settings return an error. Providing
   * starting_phase or a different billing_anchor explicitly requests replacement,
   * which resets the phase timeline, may produce billing adjustments, and does not
   * transfer addons. Custom subscriptions cannot be migrated.
   */
  @post
  @operationId("migrate-subscription")
  @summary("Migrate subscription")
  @extension(Shared.UnstableExtension, true)
  @route("/{subscriptionId}/migrate")
  migrate(@path subscriptionId: Shared.ULID, @body body: SubscriptionMigrate):
    | SubscriptionMigrateResponse
    | Common.NotFound
    | Common.Conflict
    | Common.ErrorResponses;

  /**
   * Edits a running subscription by applying an ordered batch of customizations
   * (adding or removing items, adding, removing, or stretching phases, or
   * unscheduling a pending edit). The changes may take effect immediately or at the
   * next billing cycle. Subscriptions that have add-ons cannot be edited.
   */
  @post
  @operationId("edit-subscription")
  @summary("Edit subscription")
  @extension(Shared.UnstableExtension, true)
  @route("/{subscriptionId}/edit")
  edit(@path subscriptionId: Shared.ULID, @body body: SubscriptionEdit):
    | Shared.UpdateResponse<Subscription>
    | Common.NotFound
    | Common.Conflict
    | Common.ErrorResponses;
}

/**
 * Request for updating a subscription add-on. Only the quantity can be changed;
 * the timing controls when the new quantity takes effect.
 */
@friendlyName("BillingSubscriptionAddonUpdate")
model SubscriptionAddonUpdate {
  /**
   * The new quantity of the add-on. Must be at least 0. A quantity of 0 removes the
   * add-on from the point the timing resolves to.
   */
  @summary("Quantity")
  @minValue(0)
  quantity: integer;

  /**
   * The timing of the update. A new entry is appended to the add-on's timeline at
   * this point.
   */
  @summary("Timing")
  timing: SubscriptionEditTiming;
}

interface SubscriptionAddonOperations {
  /**
   * Add add-on to a subscription.
   */
  @post
  @operationId("create-subscription-addon")
  @summary("Create a new subscription add-on")
  @extension(Shared.UnstableExtension, true)
  create(
    @path subscriptionId: Shared.ULID,
    @body request: Shared.CreateRequest<SubscriptionAddon>,
  ):
    | Shared.CreateResponse<SubscriptionAddon>
    | Common.ErrorResponses
    | Common.NotFound
    | Common.Conflict;

  /**
   * List the add-ons of a subscription.
   */
  @get
  @operationId("list-subscription-addons")
  @summary("List subscription addons")
  @extension(Shared.UnstableExtension, true)
  list(
    @path subscriptionId: Shared.ULID,
    ...Common.PagePaginationQuery,

    /**
     * Sort subscription addons returned in the response. Supported sort attributes
     * are:
     *
     * - `id`
     * - `created_at` (default)
     * - `updated_at`
     *
     * The `asc` suffix is optional as the default sort order is ascending. The `desc`
     * suffix is used to specify a descending order.
     */
    @query(#{ name: "sort" })
    sort?: Common.SortQuery,
  ):
    | Shared.PagePaginatedResponse<SubscriptionAddon>
    | Common.NotFound
    | Common.ErrorResponses;

  /**
   * Get an add-on association for a subscription.
   */
  @get
  @route("/{subscriptionAddonId}")
  @operationId("get-subscription-addon")
  @summary("Get add-on association for subscription")
  @extension(Shared.UnstableExtension, true)
  getAddon(
    @path subscriptionId: Shared.ULID,
    @path subscriptionAddonId: Shared.ULID,
  ):
    | Shared.GetResponse<SubscriptionAddon>
    | Common.ErrorResponses
    | Common.NotFound;

  /**
   * Update a subscription add-on. Only the quantity is mutable; the timing controls
   * when the new quantity takes effect. A new entry is appended to the add-on's
   * timeline.
   */
  @patch
  @route("/{subscriptionAddonId}")
  @operationId("update-subscription-addon")
  @summary("Update subscription addon")
  @extension(Shared.UnstableExtension, true)
  update(
    @path subscriptionId: Shared.ULID,
    @path subscriptionAddonId: Shared.ULID,
    @body request: SubscriptionAddonUpdate,
  ):
    | Shared.UpdateResponse<SubscriptionAddon>
    | Common.ErrorResponses
    | Common.NotFound
    | Common.Conflict;
}
