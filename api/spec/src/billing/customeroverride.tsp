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
  @operationId("listBillingProfileCustomerOverrides")
  @extension("x-internal", true)
  list(
    @query
    billingProfile?: ULID[],

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryLimitOffset,
    ...OpenMeter.QueryOrdering<CustomerOverrideOrderBy>,
  ): OpenMeter.PaginatedResponse<CustomerOverride> | OpenMeter.CommonErrors;

  /**
   * Create a new or update an existing customer override.
   */
  @post
  @route("/{customerId}")
  @operationId("upsertBillingProfileCustomerOverride")
  @summary("Create a new or update a customer override")
  @extension("x-internal", true)
  upsert(
    @path
    customerId: ULID,

    @body
    request: CustomerOverrideCreate,
  ):
    | {
        // Created
        @statusCode _: 201;

        @body body: CustomerOverride;
      }
    | {
        // Updated
        @statusCode _: 200;

        @body body: CustomerOverride;
      }
    | CustomerOverride
    | OpenMeter.NotFoundError
    | OpenMeter.CommonErrors;

  /**
   * Get a customer override by customer id.
   */
  @get
  @route("/{customerId}")
  @summary("Get a customer override")
  @operationId("getBillingProfileCustomerOverride")
  @extension("x-internal", true)
  get(
    @path
    customerId: ULID,
  ): CustomerOverride | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Delete a customer override by customer id.
   */
  @delete
  @route("/{customerId}")
  @summary("Delete a customer override")
  @operationId("deleteBillingProfileCustomerOverride")
  @extension("x-internal", true)
  delete(
    @path
    customerId: ULID,
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
  workflow?: CustomerWorkflowOverride;

  /**
   * The billing profile this override is associated with.
   *
   * If not provided, the default billing profile is chosen if available.
   */
  billingProfile?: ULID;
}

/**
 * Payload for creating a new or updating an existing customer override.
 */
@friendlyName("BillingCustomerOverrideCreate")
model CustomerOverrideCreate {
  workflow?: Rest.Resource.ResourceCreateModel<CustomerWorkflowOverride>;

  /**
   * The billing profile this override is associated with.
   *
   * If not provided, the default billing profile is chosen if available.
   */
  billingProfile?: ULID;
}

/**
 * Customer specific workflow overrides.
 */
@friendlyName("BillingCustomerWorkflowOverride")
model CustomerWorkflowOverride {
  ...BillingWorkflow;

  // Note: these are only available for read, as provider override is not supported via the customer override.
  // to do that the customer should be assocatied with a new billing profile instead.

  @visibility("read")
  taxApp: OpenMeter.App.App;

  /*
   * The invoicing app used for this workflow
   */
  @visibility("read")
  invoicingApp: OpenMeter.App.App;

  /*
   * The payment app used for this workflow
   */
  @visibility("read")
  paymentApp: OpenMeter.App.App;
}
