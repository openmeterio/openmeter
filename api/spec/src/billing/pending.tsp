import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter.Billing;

@route("/api/v1/billing/pending")
interface PendingCharges {
  /**
   * List pending charges
   */
  @get
  @summary("List pending charges")
  get(): Invoice[] | OpenMeter.CommonErrors;

  /**
   * Create a new pending charge
   */
  @post
  @route("/item")
  @summary("Create a pending charge")
  create(@body request: InvoiceItem[]): {
    @statusCode _: 201;
    @body body: InvoiceItem[];
  } | OpenMeter.CommonErrors;

  /**
   * Delete a pending charge
   */
  @delete
  @route("/item/:id")
  @summary("Delete a pending charge")
  delete(@path id: ULID): {
    @statusCode _: 204;

    // TODO: align delete repsonse!!
    @body body: InvoiceItem;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}
