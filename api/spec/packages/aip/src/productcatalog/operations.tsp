import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./ratecard.tsp";
import "./plan.tsp";
import "./planaddon.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace ProductCatalog;

/**
 * Filter options for listing plans.
 */
@friendlyName("ListPlansParamsFilter")
model ListPlansParamsFilter {
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  key?: Common.StringFieldFilter;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  name?: Common.StringFieldFilter;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  status?: Common.StringFieldFilter;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  currency?: Common.StringFieldFilter;
}

interface PlanOperations {
  /**
   * List all plans.
   */
  @get
  @operationId("list-plans")
  @summary("List plans")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  listPlans(
    ...Common.PagePaginationQuery,

    /**
     * Sort plans returned in the response.
     * Supported sort attributes are:
     * - `id`
     * - `key`
     * - `version`
     * - `created_at` (default)
     * - `updated_at`
     */
    @query(#{ name: "sort" })
    sort?: Common.SortQuery,

    /**
     * Filter plans returned in the response.
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListPlansParamsFilter,
  ): Shared.PagePaginatedResponse<Plan> | Common.ErrorResponses;

  /**
   * Create a new plan.
   */
  @post
  @operationId("create-plan")
  @summary("Create a new plan.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  createPlan(
    @body plan: Shared.CreateRequest<Plan>,
  ): Shared.CreateResponse<Plan> | Common.ErrorResponses;

  /**
   * Update a plan by id.
   */
  @put
  @route("/{planId}")
  @operationId("update-plan")
  @summary("Update a plan by id.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  updatePlan(
    @path planId: Shared.ULID,
    @body plan: Shared.UpsertRequest<Plan>,
  ):
    | Shared.UpsertResponse<Plan>
    | Common.ErrorResponses
    | Common.Gone
    | Common.NotFound;

  /**
   * Get a plan by id.
   */
  @get
  @route("/{planId}")
  @operationId("get-plan")
  @summary("Get a plan by id.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  getPlan(@path planId: Shared.ULID):
    | Shared.GetResponse<Plan>
    | Common.ErrorResponses
    | Common.Gone
    | Common.NotFound;

  /**
   * Delete a plan by id.
   */
  @delete
  @route("/{planId}")
  @operationId("delete-plan")
  @summary("Delete a plan by id.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  deletePlan(
    @path planId: Shared.ULID,
  ): Shared.DeleteResponse | Common.ErrorResponses | Common.NotFound;

  /**
   * Archive a plan version.
   */
  @post
  @route("/{planId}/archive")
  @operationId("archive-plan")
  @summary("Archive a plan version.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  archivePlan(
    @path planId: Shared.ULID,
  ): Shared.UpdateResponse<Plan> | Common.ErrorResponses | Common.NotFound;

  /**
   * Publish a plan version.
   */
  @post
  @route("/{planId}/publish")
  @operationId("publish-plan")
  @summary("Publish a plan version.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  publishPlan(
    @path planId: Shared.ULID,
  ): Shared.UpdateResponse<Plan> | Common.ErrorResponses | Common.NotFound;
}

interface AddonOperations {
  /**
   * List all add-ons.
   */
  @get
  @operationId("list-addons")
  @summary("List all add-ons.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  listAddons(
    ...Common.PagePaginationQuery,
  ): Shared.PagePaginatedResponse<Addon> | Common.ErrorResponses;

  /**
   * Create a new add-on.
   */
  @post
  @operationId("create-addon")
  @summary("Create a new add-on.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  createAddon(
    @body addon: Shared.CreateRequest<Addon>,
  ): Shared.CreateResponse<Addon> | Common.ErrorResponses;

  /**
   * Update an add-on by id.
   */
  @put
  @route("/{addonId}")
  @operationId("update-addon")
  @summary("Update an add-on by id.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  updateAddon(
    @path addonId: Shared.ULID,
    @body addon: Shared.UpsertRequest<Addon>,
  ):
    | Shared.UpsertResponse<Addon>
    | Common.ErrorResponses
    | Common.Gone
    | Common.NotFound;

  /**
   * Get add-on by id.
   */
  @get
  @route("/{addonId}")
  @operationId("get-addon")
  @summary("Get add-on by id.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  getAddon(@path addonId: Shared.ULID):
    | Shared.GetResponse<Addon>
    | Common.ErrorResponses
    | Common.Gone
    | Common.NotFound;

  /**
   * Soft delete add-on by id.
   */
  @delete
  @route("/{addonId}")
  @operationId("delete-addon")
  @summary("Soft delete add-on by id.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  deleteAddon(
    @path addonId: Shared.ULID,
  ): Shared.DeleteResponse | Common.ErrorResponses | Common.NotFound;

  /**
   * Archive an add-on version.
   */
  @post
  @route("/{addonId}/archive")
  @operationId("archive-addon")
  @summary("Archive an add-on version.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  archiveAddon(
    @path addonId: Shared.ULID,
  ): Shared.UpdateResponse<Addon> | Common.ErrorResponses | Common.NotFound;

  /**
   * Publish an add-on version.
   */
  @post
  @route("/{addonId}/publish")
  @operationId("publish-addon")
  @summary("Publish an add-on version.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  publishAddon(
    @path addonId: Shared.ULID,
  ): Shared.UpdateResponse<Addon> | Common.ErrorResponses | Common.NotFound;
}

interface PlanAddonOperations {
  /**
   * List add-ons associated with a plan.
   */
  @get
  @operationId("list-plan-addons")
  @summary("List add-ons for a plan.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  listPlanAddons(
    @path planId: Shared.ULID,
    ...Common.PagePaginationQuery,
  ): Shared.PagePaginatedResponse<PlanAddon> | Common.ErrorResponses | Common.NotFound;

  /**
   * Add an add-on to a plan.
   */
  @post
  @operationId("create-plan-addon")
  @summary("Add an add-on to a plan.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  createPlanAddon(
    @path planId: Shared.ULID,
    @body plan_addon: Shared.CreateRequest<PlanAddon>,
  ): Shared.CreateResponse<PlanAddon> | Common.ErrorResponses | Common.NotFound;

  /**
   * Get an add-on association for a plan.
   */
  @get
  @route("/{planAddonId}")
  @operationId("get-plan-addon")
  @summary("Get an add-on association for a plan.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  getPlanAddon(
    @path planId: Shared.ULID,
    @path planAddonId: Shared.ULID,
  ): Shared.GetResponse<PlanAddon> | Common.ErrorResponses | Common.NotFound;

  /**
   * Update an add-on association for a plan.
   */
  @put
  @route("/{planAddonId}")
  @operationId("update-plan-addon")
  @summary("Update an add-on association for a plan.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  updatePlanAddon(
    @path planId: Shared.ULID,
    @path planAddonId: Shared.ULID,
    @body plan_addon: Shared.UpsertRequest<PlanAddon>,
  ): Shared.UpsertResponse<PlanAddon> | Common.ErrorResponses | Common.NotFound;

  /**
   * Remove an add-on from a plan.
   */
  @delete
  @route("/{planAddonId}")
  @operationId("delete-plan-addon")
  @summary("Remove an add-on from a plan.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  deletePlanAddon(
    @path planId: Shared.ULID,
    @path planAddonId: Shared.ULID,
  ): Shared.DeleteResponse | Common.ErrorResponses | Common.NotFound;
}
