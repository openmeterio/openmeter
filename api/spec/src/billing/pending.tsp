import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter.Billing;

@route("/api/v1/billing/pending")
@tag("Billing (Experimental)")
interface PendingCharges {
  /**
   * List pending charges
   */
  @get
  @summary("List pending charges")
  getPendingCharges(): GOBL.Bill.Invoice[] | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Create a new pending charge
   *
   * Note: Manually added charges can only be static, one-time charges. They cannot be based on a meter.
   */
  @post
  @route("/item")
  @summary("Create a pending charge")
  create(@body request: GOBL.Bill.InvoiceItem[]): {
    @statusCode _: 201;
    @body body: GOBL.Bill.InvoiceItem[];
  } | OpenMeter.CommonErrors;

  /**
   * Get a pending charge by ID
   */
  @get
  @route("/item/:id")
  @summary("Get a pending charge")
  getItem(@path id: ULID): GOBL.Bill.InvoiceItem | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Delete a pending charge
   */
  @delete
  @route("/item/:id")
  @summary("Delete a pending charge")
  delete(@path id: ULID): {
    @statusCode _: 200;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}
