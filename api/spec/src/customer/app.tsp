import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

import "../app";
import "../rest.tsp";
import "../errors.tsp";
import "../types.tsp";
import "../query.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.Customer;

/**
 * Customer App Data API.
 */
@route("/api/v1/customers/{customerIdOrKey}/apps")
@tag("CustomersApps")
@friendlyName("CustomerApps")
interface CustomersApps {
  /**
   * List customers app data.
   */
  @get
  @operationId("listCustomerAppData")
  @summary("List customer app data")
  listAppData(
    @path customerIdOrKey: ULIDOrExternalKey,
    ...ListCustomerAppDataParams,
  ): PaginatedResponse<App.CustomerAppData> | NotFoundError | CommonErrors;

  /**
   * Upsert customer app data.
   */
  @put
  @operationId("upsertCustomerAppData")
  @summary("Upsert customer app data")
  upsertAppData(
    @path customerIdOrKey: ULIDOrExternalKey,
    @body appData: App.CustomerAppData[],
  ): App.CustomerAppData[] | NotFoundError | CommonErrors;

  /**
   * Delete customer app data.
   */
  @delete
  @route("/{appId}")
  @operationId("deleteCustomerAppData")
  @summary("Delete customer app data")
  deleteAppData(@path customerIdOrKey: ULIDOrExternalKey, @path appId: ULID): {
    @statusCode _: 204;
  } | NotFoundError | CommonErrors;
}

/**
 * Query params for listing customers app data.
 */
@friendlyName("listCustomerAppDataParams")
model ListCustomerAppDataParams {
  ...QueryPagination;

  /**
   * Filter customer data by app type.
   */
  @query
  @example(App.Type.Stripe)
  type?: App.Type;
}
