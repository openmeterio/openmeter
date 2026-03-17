import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./ratecard.tsp";
import "./plan.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace ProductCatalog;

interface ProductCatalogOperations {
  /**
   *  List all plans.
   */
  @get
  @route("/plans")
  @operationId("list-plans")
  @summary("List plans")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  listPlans(
    ...Common.PagePaginationQuery,
  ): Shared.PagePaginatedResponse<Plan> | Common.ErrorResponses;

  /**
   *  Create a new plan.
   */
  @post
  @route("/plans")
  @operationId("create-plan")
  @summary("Create a new plan.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  createPlan(
    @body
    plan: Shared.CreateRequest<Plan>,
  ): Shared.CreateResponse<Plan> | Common.ErrorResponses;

  /**
   *  Upsert a plan by id.
   */
  @put
  @route("/plans/{planId}")
  @operationId("upsert-plan")
  @summary("Upsert a plan by id.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  upsertPlan(
    @path planId: Shared.ULID,

    @body
    plan: Shared.UpsertRequest<Plan>,
  ):
    | Shared.UpsertResponse<Plan>
    | Common.ErrorResponses
    | Common.Gone
    | Common.NotFound;

  /**
   *  Get a plan by id.
   */
  @get
  @route("/plans/{planId}")
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
   * Delete plan by id.
   */
  @delete
  @route("/plans/{planId}")
  @operationId("delete-plan")
  @summary("Delete a plan by id.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  deletePlan(
    @path planId: Shared.ULID,
  ): Shared.DeleteResponse | Common.ErrorResponses | Common.NotFound;

  /**
   *  List all add-ons.
   */
  @get
  @route("/addons")
  @operationId("list-addons")
  @summary("List all add-ons.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  listAddons(
    ...Common.PagePaginationQuery,
  ): Shared.PagePaginatedResponse<Addon> | Common.ErrorResponses;

  /**
   *  List all add-ons.
   */
  @post
  @route("/addons")
  @operationId("create-addon")
  @summary("Create a new add-on")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  createAddon(
    addon: Shared.CreateRequest<Addon>,
  ): Shared.CreateResponse<Addon> | Common.ErrorResponses;

  /**
   *  Upsert add-on by id.
   */
  @put
  @route("/addons/{addonId}")
  @operationId("upsert-addon")
  @summary("Upsert add-on by id")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  upsertAddon(
    @path addonId: Shared.ULID,

    @body
    addon: Shared.UpsertRequest<Addon>,
  ):
    | Shared.UpsertResponse<Addon>
    | Common.ErrorResponses
    | Common.Gone
    | Common.NotFound;

  /**
   *  Get add-on by id.
   */
  @get
  @route("/addons/{addonId}")
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
  @route("/addons/{addonId}")
  @operationId("delete-addon")
  @summary("Soft delete add-on by id.")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  deleteAddon(
    @path addonId: Shared.ULID,
  ): Shared.DeleteResponse | Common.ErrorResponses | Common.NotFound;
}
