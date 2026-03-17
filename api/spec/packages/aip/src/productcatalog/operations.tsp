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
  list(
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
  create(
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
  upsert(
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
  get(@path planId: Shared.ULID):
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
  delete(
    @path planId: Shared.ULID,
  ): Shared.DeleteResponse | Common.ErrorResponses | Common.NotFound;
}
