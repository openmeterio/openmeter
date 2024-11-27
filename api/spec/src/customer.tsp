import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

import "./rest.tsp";
import "./errors.tsp";
import "./pagination.tsp";
import "./types.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;

/**
 * Customer API.
 */
@route("/api/v1/customers")
@tag("Customers")
interface Customers {
  /**
   * Create a new customer.
   */
  @post
  @operationId("createCustomer")
  @summary("Create customer")
  @extension("x-internal", true)
  create(@body customer: Rest.Resource.ResourceCreateModel<Customer>): {
    @statusCode _: 201;
    @body body: Customer;
  } | CommonErrors;

  /**
   * List customers.
   */
  @get
  @operationId("listCustomers")
  @summary("List customers")
  @extension("x-internal", true)
  list(...ListCustomersParams): PaginatedResponse<Customer> | CommonErrors;

  /**
   * Get a customer by ID.
   */
  @get
  @route("/{id}")
  @operationId("getCustomer")
  @summary("Get customer")
  @extension("x-internal", true)
  get(@path id: ULID): Customer | NotFoundError | CommonErrors;

  /**
   * Update a customer by ID.
   */
  @put
  @route("/{id}")
  @operationId("updateCustomer")
  @summary("Update customer")
  @extension("x-internal", true)
  update(
    @path id: ULID,
    @body customer: Rest.Resource.ResourceReplaceModel<Customer>,
  ): Customer | NotFoundError | CommonErrors;

  /**
   * Delete a customer by ID.
   */
  @delete
  @route("/{id}")
  @operationId("deleteCustomer")
  @summary("Delete customer")
  @extension("x-internal", true)
  delete(@path id: ULID): {
    @statusCode _: 204;
  } | NotFoundError | CommonErrors;

  /**
   * List customers app data.
   */
  @get
  @route("/{customerId}/apps")
  @operationId("listCustomerAppData")
  @summary("List customer app data")
  @extension("x-internal", true)
  listAppData(
    ...ListCustomerAppDataParams,
  ): PaginatedResponse<App.CustomerAppData> | NotFoundError | CommonErrors;

  /**
   * Upsert customer app data.
   */
  @put
  @route("/{customerId}/apps")
  @operationId("upsertCustomerAppData")
  @summary("Upsert customer app data")
  @extension("x-internal", true)
  upsertAppData(
    @path customerId: ULID,
    @body appData: App.CustomerAppData[],
  ): App.CustomerAppData[] | NotFoundError | CommonErrors;

  /**
   * Delete customer app data.
   */
  @delete
  @route("/{customerId}/apps/{appId}")
  @operationId("deleteCustomerAppData")
  @summary("Delete customer app data")
  @extension("x-internal", true)
  deleteAppData(@path customerId: ULID, @path appId: ULID): {
    @statusCode _: 204;
  } | NotFoundError | CommonErrors;
}

/**
 * Query params for listing customers.
 */
@friendlyName("queryCustomerList")
model ListCustomersParams {
  ...PaginatedQuery;

  /**
   * Order customers by.
   * @TODO: set default value to CustomerOrderBy.name
   */
  ...OpenMeter.QueryOrdering<CustomerOrderBy>;

  /**
   * Include deleted customers.
   */
  @query
  includeDeleted?: boolean = false;

  /**
   * Filter customers by name.
   * Case-insensitive partial match.
   */
  @query
  @example("ACME")
  name?: string;

  /**
   * Filter customers by primary email.
   * Case-insensitive partial match.
   */
  @query
  @example("acme@test.com")
  primaryEmail?: string;

  /**
   * Filter customers by usage attribution subject.
   * Case-insensitive partial match.
   */
  @query
  @example("my_subject_key")
  subject?: string;
}

/**
 * Query params for listing customers.
 */
@friendlyName("queryCustomerList")
model ListCustomerAppDataParams {
  ...PaginatedQuery;

  /**
   * Customer ID.
   */
  @path
  customerId: ULID;

  /**
   * Filter customer data by app type.
   */
  @query
  @example(App.Type.Stripe)
  type?: App.Type;
}

/**
 * Order by options for customers.
 */
@friendlyName("CustomerOrderBy")
enum CustomerOrderBy {
  id: "id",
  name: "name",
  createdAt: "createdAt",
}

/**
 * A customer object.
 */
@friendlyName("Customer")
@example(#{
  id: "01G65Z755AFWAKHE12NY0CQ9FH",
  name: "ACME Inc.",
  usageAttribution: #{ subjectKeys: #["my_subject_key"] },
  createdAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
  updatedAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
})
model Customer {
  ...Resource;

  /**
   * Timezone of the customer.
   *
   * @TODO: use a Timezone type to validate the value
   */
  @summary("Timezone")
  timezone?: string;

  /**
   * Mapping to attribute metered usage to the customer
   */
  @summary("Usage Attribution")
  usageAttribution: CustomerUsageAttribution;

  /**
   * The primary email address of the customer.
   */
  @summary("Primary Email")
  primaryEmail?: string;

  /**
   * Currency of the customer.
   * Used for billing, tax and invoicing.
   */
  @summary("Currency")
  currency?: CurrencyCode;

  /**
   * The billing address of the customer.
   * Used for tax and invoicing.
   */
  @summary("Billing Address")
  billingAddress?: Address;
}

/**
 * Mapping to attribute metered usage to the customer.
 * One customer can have multiple subjects,
 * but one subject can only belong to one customer.
 */
@friendlyName("CustomerUsageAttribution")
model CustomerUsageAttribution {
  /**
   * The subjects that are attributed to the customer.
   * @TODO allow multiple subjects
   */
  @summary("SubjectKeys")
  @minItems(1)
  @maxItems(1)
  subjectKeys: string[];
}
