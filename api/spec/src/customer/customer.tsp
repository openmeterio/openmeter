import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

import "../productcatalog";
import "../rest.tsp";
import "../errors.tsp";
import "../types.tsp";
import "../query.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;

/**
 * Customer API.
 */
@route("/api/v1/customers")
@tag("Customers")
@friendlyName("Customers")
interface CustomersEndpoints {
  /**
   * Create a new customer.
   */
  @post
  @operationId("createCustomer")
  @summary("Create customer")
  create(
    @body customer: TypeSpec.Rest.Resource.ResourceCreateModel<Customer>,
  ): {
    @statusCode _: 201;
    @body body: Customer;
  } | CommonErrors;

  /**
   * List customers.
   */
  @get
  @operationId("listCustomers")
  @summary("List customers")
  list(...ListCustomersParams): PaginatedResponse<Customer> | CommonErrors;

  /**
   * Get a customer by ID or key.
   */
  @get
  @route("/{customerIdOrKey}")
  @operationId("getCustomer")
  @summary("Get customer")
  get(
    @path customerIdOrKey: ULIDOrExternalKey,
    ...GetCustomerParams,
  ): Customer | NotFoundError | CommonErrors;

  /**
   * Update a customer by ID.
   */
  @put
  @route("/{customerIdOrKey}")
  @operationId("updateCustomer")
  @summary("Update customer")
  update(
    @path customerIdOrKey: ULIDOrExternalKey,
    @body customer: TypeSpec.Rest.Resource.ResourceReplaceModel<Customer>,
  ): Customer | NotFoundError | CommonErrors;

  /**
   * Delete a customer by ID.
   */
  @delete
  @route("/{customerIdOrKey}")
  @operationId("deleteCustomer")
  @summary("Delete customer")
  delete(@path customerIdOrKey: ULIDOrExternalKey): {
    @statusCode _: 204;
  } | NotFoundError | CommonErrors;

  /**
   * Lists all subscriptions for a customer.
   */
  @get
  @route("/{customerIdOrKey}/subscriptions")
  @operationId("listCustomerSubscriptions")
  @summary("List customer subscriptions")
  listCustomerSubscriptions(
    @path customerIdOrKey: ULIDOrExternalKey,
    @query(#{ explode: true }) status?: SubscriptionStatus[],
    ...QueryOrdering<CustomerSubscriptionOrderBy>,
    ...QueryPagination,
  ): PaginatedResponse<Subscription> | NotFoundError | CommonErrors;
}

/**
 * Order by options for customer subscriptions.
 */
@friendlyName("CustomerSubscriptionOrderBy")
enum CustomerSubscriptionOrderBy {
  activeFrom: "activeFrom",
  activeTo: "activeTo",
}

/**
 * Query params for get customer.
 */
@friendlyName("queryCustomerGet")
model GetCustomerParams {
  /**
   * What parts of the customer output to expand
   */
  @query(#{ explode: true })
  expand?: CustomerExpand[];
}

/**
 * Query params for listing customers.
 */
@friendlyName("queryCustomerList")
model ListCustomersParams {
  ...QueryPagination;

  /**
   * Order customers by.
   * @TODO: set default value to CustomerOrderBy.name
   */
  ...QueryOrdering<CustomerOrderBy>;

  /**
   * Include deleted customers.
   */
  @query
  includeDeleted?: boolean = false;

  /**
   * Filter customers by key.
   * Case-insensitive partial match.
   */
  @query
  key?: string;

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

  /**
   * Filter customers by the plan ID of their subscription.
   * Mutually exclusive with planKey.
   */
  @query
  @example("01KCZXM1TB1W4VS3HMACN0B5P5")
  plan?: ULID;

  /**
   * Filter customers by the plan key of their subscription.
   * Case sensitive exact match of the plan key.
   * Mutually exclusive with plan.
   */
  @query
  @example("pro_plan")
  planKey?: string;

  /**
   * What parts of the list output to expand in listings
   */
  @query(#{ explode: true })
  expand?: CustomerExpand[];
}

/**
 * CustomerExpand specifies the parts of the customer to expand in the list output.
 */
@friendlyName("CustomerExpand")
enum CustomerExpand {
  subscriptions: "subscriptions",
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
   * An optional unique key of the customer.
   * Either key or usageAttribution.subjectKeys must be provided.
   * Useful to reference the customer in external systems.
   * For example, your database ID.
   */
  @summary("Key")
  key?: ExternalKey;

  /**
   * Mapping to attribute metered usage to the customer
   * Either key or usageAttribution.subjectKeys must be provided.
   */
  @summary("Usage Attribution")
  usageAttribution?: CustomerUsageAttribution;

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

  /**
   * The ID of the Subscription if the customer has one.
   */
  @summary("Current Subscription ID")
  @visibility(Lifecycle.Read)
  currentSubscriptionId?: ULID;

  /**
   * The subscriptions of the customer.
   * Only with the `subscriptions` expand option.
   */
  @summary("Subscriptions")
  @visibility(Lifecycle.Read)
  subscriptions?: Subscription[];

  /**
   * Set of key-value pairs managed by the system. Cannot be modified by user.
   */
  @summary("Annotations")
  @visibility(Lifecycle.Read)
  annotations?: Annotations;
}

/**
 * Mapping to attribute metered usage to the customer.
 * One customer can have zero or more subjects,
 * but one subject can only belong to one customer.
 */
@friendlyName("CustomerUsageAttribution")
model CustomerUsageAttribution {
  /**
   * The subjects that are attributed to the customer.
   * Can be empty when no subjects are associated with the customer.
   */
  @summary("SubjectKeys")
  @minItems(0)
  subjectKeys: SubjectKey[];
}

/**
 * SubjectKey is a key that is used to identify a subject.
 */
@minLength(1)
@extension("x-inline", true)
scalar SubjectKey extends string;
