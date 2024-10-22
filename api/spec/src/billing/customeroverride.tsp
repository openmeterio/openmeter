using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.Billing;

@route("/api/v1/billing/customer")
@tag("Billing (Experimental)")
interface CustomerOverrides {
  /**
   * List customer overrides
   */
  @get
  @summary("List customer overrides")
  @operationId("billingListCustomerOverrides")
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
  @route("/{customerId}")
  @summary("Create/update a customer override")
  @operationId("billingUpsertCustomerOverride")
  upsert(
    @path
    customerId: ULID,

    @body
    request: CustomerWorkflowOverride,
  ):
    | {
        // Updated
        @statusCode _: 200;

        @body body: CustomerOverride;
      }
    | {
        // Created
        @statusCode _: 201;

        @body body: CustomerOverride;
      }
    | OpenMeter.NotFoundError
    | OpenMeter.CommonErrors;

  /**
   * Get a customer override by id.
   */
  @get
  @route("/{customerId}")
  @summary("Get a customer override")
  @operationId("billingGetCustomerOverrideById")
  get(
    @path
    customerId: ULID,
  ): CustomerOverride | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Delete a customer override by id.
   */
  @delete
  @route("/{customerId}")
  @summary("Delete a customer override")
  @operationId("billingDeleteCustomerOverride")
  delete(
    @path
    customerId: ULIDOrKey,
  ): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

/**
 * Order by options for customers.
 */
@friendlyName("BillingCustomerOverrideOrderBy")
enum CustomerOverrideOrderBy {
  id: "id",
}

/**
 * Customer specific workflow overrides.
 */
@friendlyName("BillingCustomerOverride")
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

@friendlyName("BillingCustomerWorkflowOverride")
model CustomerWorkflowOverride {
  ...OmitProperties<Workflow, "taxApp" | "invoicingApp" | "paymentApp">;

  // Note: these are only available for read, as provider override is not supported via the customer override.
  // to do that the customer should be assocatied with a new billing profile instead.

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
