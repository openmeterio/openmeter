using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.Billing;

@route("/api/v1/billing/customers")
@tag("Billing (Beta)")
interface CustomerOverrides {
  /**
   * List customer overrides
   */
  @get
  @summary("List customer overrides")
  @operationId("listBillingProfileCustomerOverrides")
  list(
    @query(#{ explode: true })
    billingProfile?: ULID[],

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryLimitOffset,
    ...OpenMeter.QueryOrdering<BillingProfileCustomerOverrideOrderBy>,
  ): OpenMeter.PaginatedResponse<BillingProfileCustomerOverride> | OpenMeter.CommonErrors;

  /**
   * Create a new or update an existing customer override.
   */
  @post
  @route("/{customerId}")
  @operationId("upsertBillingProfileCustomerOverride")
  @summary("Create a new or update a customer override")
  upsert(
    @path
    customerId: ULID,

    @body
    request: BillingProfileCustomerOverrideCreate,
  ):
    | {
        // Created
        @statusCode _: 201;

        @body body: BillingProfileCustomerOverride;
      }
    | {
        // Updated
        @statusCode _: 200;

        @body body: BillingProfileCustomerOverride;
      }
    | BillingProfileCustomerOverride
    | OpenMeter.NotFoundError
    | OpenMeter.CommonErrors;

  /**
   * Get a customer override by customer id.
   */
  @get
  @route("/{customerId}")
  @summary("Get a customer override")
  @operationId("getBillingProfileCustomerOverride")
  get(
    @path
    customerId: ULID,
  ): BillingProfileCustomerOverride | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Delete a customer override by customer id.
   */
  @delete
  @route("/{customerId}")
  @summary("Delete a customer override")
  @operationId("deleteBillingProfileCustomerOverride")
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
@friendlyName("BillingProfileCustomerOverrideOrderBy")
enum BillingProfileCustomerOverrideOrderBy {
  id: "id",
}

/**
 * Customer specific workflow overrides.
 */
@friendlyName("BillingProfileCustomerOverride")
model BillingProfileCustomerOverride {
  ...ResourceTimestamps;

  /**
   * The details of the overrides active for this specific customer.
   */
  workflow?: BillingProfileCustomerWorkflowOverride;

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
@friendlyName("BillingProfileCustomerOverrideCreate")
model BillingProfileCustomerOverrideCreate {
  /**
   * The details of the overrides active for this specific customer.
   */
  workflow?: Rest.Resource.ResourceCreateModel<BillingProfileCustomerWorkflowOverride>;

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
@friendlyName("BillingProfileCustomerWorkflowOverride")
model BillingProfileCustomerWorkflowOverride {
  ...BillingWorkflow;

  // Note: these are only available for read, as provider override is not supported via the customer override.
  // to do that the customer should be assocatied with a new billing profile instead.

  /**
   * The tax app used for this workflow
   */
  @visibility(Lifecycle.Read)
  taxApp: OpenMeter.App.App;

  /**
   * The invoicing app used for this workflow
   */
  @visibility(Lifecycle.Read)
  invoicingApp: OpenMeter.App.App;

  /**
   * The payment app used for this workflow
   */
  @visibility(Lifecycle.Read)
  paymentApp: OpenMeter.App.App;
}
