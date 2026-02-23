import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./codes.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Tax;

interface TaxCodesOperations {
  @post
  @operationId("create-tax-code")
  @summary("Create tax code")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  create(
    @body
    tax_code: Shared.CreateRequest<TaxCode>,
  ): Shared.CreateResponse<TaxCode> | Common.ErrorResponses;

  @get
  @operationId("get-tax-code")
  @summary("Get tax code")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  get(
    @path taxCodeId: Shared.ULID,
  ): Shared.GetResponse<TaxCode> | Common.NotFound | Common.ErrorResponses;

  @get
  @operationId("list-tax-codes")
  @summary("List tax codes")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  list(
    ...Common.PagePaginationQuery,
  ): Shared.PagePaginatedResponse<TaxCode> | Common.ErrorResponses;

  @put
  @operationId("upsert-tax-code")
  @summary("Upsert tax code")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  upsert(
    @path taxCodeId: Shared.ULID,

    @body
    tax_code: Shared.UpsertRequest<TaxCode>,
  ):
    | Shared.UpsertResponse<TaxCode>
    | Common.Gone
    | Common.NotFound
    | Common.ErrorResponses;

  @delete
  @operationId("delete-tax-code")
  @summary("Delete tax code")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  delete(
    @path taxCodeId: Shared.ULID,
  ): Shared.DeleteResponse | Common.NotFound | Common.ErrorResponses;
}
