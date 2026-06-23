import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "../billing/profile.tsp";
import "./invoice.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Invoices;

/**
 * Operations for managing billing invoices.
 */
interface InvoicesOperations {
  /**
   * Get a billing invoice by ID.
   *
   * Returns the full invoice resource including line items, status details, totals,
   * and workflow configuration snapshot.
   */
  @get
  @operationId("get-invoice")
  @summary("Get a billing invoice")
  get(@path invoiceId: Shared.ULID):
    | Shared.GetResponse<BillingInvoice>
    | Common.NotFound
    | Common.ErrorResponses;
}
