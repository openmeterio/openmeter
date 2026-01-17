import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./profile.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Billing;

interface BillingProfilesOperations {
  /**
   * List billing profiles.
   */
  @get
  @operationId("list-billing-profiles")
  @summary("List billing profiles")
  @extension(Shared.InternalExtension, true)
  @extension(Shared.UnstableExtension, true)
  list(
    ...Common.PagePaginationQuery,
  ): Shared.PagePaginatedResponse<BillingProfile> | Common.ErrorResponses;

  /**
   * Create a new billing profile.
   *
   * Billing profiles contain the settings for billing and controls invoice generation.
   * An organization can have multiple billing profiles defined.
   * A billing profile is linked to a specific app. This association is established during the billing profile's creation and remains immutable.
   */
  @post
  @operationId("create-billing-profile")
  @summary("Create a new billing profile")
  @extension(Shared.InternalExtension, true)
  @extension(Shared.UnstableExtension, true)
  create(
    @body profile: Shared.CreateRequest<BillingProfile>,
  ): Shared.CreateResponse<BillingProfile> | Common.ErrorResponses;

  /**
   * Get a billing profile.
   */
  @get
  @operationId("get-billing-profile")
  @summary("Get a billing profile")
  @extension(Shared.InternalExtension, true)
  @extension(Shared.UnstableExtension, true)
  get(
    @path id: Shared.ULID,
  ): Shared.GetResponse<BillingProfile> | Common.NotFound | Common.ErrorResponses;

  /**
   * Update a billing profile.
   */
  @put
  @operationId("update-billing-profile")
  @summary("Update a billing profile")
  @extension(Shared.InternalExtension, true)
  @extension(Shared.UnstableExtension, true)
  update(
    @path id: Shared.ULID,
    @body profile: Shared.UpsertRequest<BillingProfile>,
  ): Shared.UpdateResponse<BillingProfile> | Common.NotFound | Common.ErrorResponses;
  /**
   * Delete a billing profile.
   *
   * Only such billing profiles can be deleted that are:
   * - not the default profile
   * - not pinned to any customer using customer overrides
   * - only have finalized invoices
   */
  @delete
  @operationId("delete-billing-profile")
  @summary("Delete a billing profile")
  @extension(Shared.InternalExtension, true)
  @extension(Shared.UnstableExtension, true)
  delete(
    @path id: Shared.ULID,
  ): Shared.DeleteResponse | Common.NotFound | Common.ErrorResponses;
}
