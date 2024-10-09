import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter.Billing;

@route("/api/v1/billing/customer")
@tag("Billing (Experimental)")
interface Customers {
  /**
   * List customer overrides
   */
  @get
  @summary("List customer overrides")
  @operationId("listCustomerOverrides")
  list(
    @query
    billingProfile?: Array<ULID>,

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryLimitOffset,
    ...OpenMeter.QueryOrdering<CustomerOverrideOrderBy>,
  ): OpenMeter.PaginatedResponse<CustomerOverride> | OpenMeter.CommonErrors;

  /**
   * Create/update a new customer override.
   */
  @post
  @route("/:customerIdOrKey")
  @summary("Create/update a customer override")
  @operationId("createCustomerOverride")
  create(
    @path
    customerIdOrKey: ULIDOrKey,

    @body
    request: CustomerWorkflowOverride,
  ): {
    @statusCode _: 200;
    @body body: CustomerOverride;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Get a customer override by id.
   */
  @get
  @route("/:customerIdOrKey")
  @summary("Get a customer override")
  @operationId("getCustomerOverrideById")
  get(
    @path
    customerIdOrKey: ULIDOrKey,
  ): CustomerOverride | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Delete a customer override by id.
   */
  @delete
  @route("/:customerIdOrKey")
  @summary("Delete a customer override")
  @operationId("deleteCustomerOverride")
  delete(
    @path
    customerIdOrKey: ULIDOrKey,
  ): {
    @statusCode _: 200;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

/**
 * Order by options for customers.
 */
@friendlyName("CustomerOrderBy")
enum CustomerOverrideOrderBy {
  id: "id",
  created: "created",
  updated: "updated",
}

model CustomerOverride {
  ...ResourceTimestamps;
  workflow: CustomerWorkflowOverride;
  /**
   * The billing profile this override is associated with.
   *
   * If not provided, the default billing profile is chosen if available.
   */
  billingProfile?: ULID;
}

model CustomerWorkflowOverride {
  ...OmitProperties<Workflow, "taxApp" | "invoicingApp" | "paymentApp">;

  @summary("The tax app used for this workflow")
  @visibility("read", "query")
  taxApp: OpenMeter.App.App;

  @summary("The invoicing app used for this workflow")
  @visibility("read", "query")
  invoicingApp: OpenMeter.App.App;

  @summary("The payment app used for this workflow")
  @visibility("read", "query")
  paymentApp: OpenMeter.App.App;
}
