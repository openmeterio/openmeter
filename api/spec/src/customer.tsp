import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

import "./errors.tsp";
import "./billing.tsp";
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
  create(@body customer: Customer): Customer | CommonErrors;

  /**
   * List customers.
   */
  @get
  @operationId("listCustomers")
  list(...ListCustomersParams): Paginated<Customer>[] | CommonErrors;

  /**
   * Get a customer by ID or key.
   */
  @get
  @route("/{customerIdOrKey}")
  @operationId("getCustomer")
  get(@path customerIdOrKey: CustomerIdentifier): Customer | NotFoundError | CommonErrors;

  /**
   * Update a customer by ID or key.
   */
  @put
  @route("/{customerIdOrKey}")
  @operationId("updateCustomer")
  update(@path customerIdOrKey: CustomerIdentifier, @body customer: Customer): Customer | NotFoundError | CommonErrors;

  /**
   * Delete a customer by ID or key.
   */
  @delete
  @route("/{customerIdOrKey}")
  @operationId("deleteCustomer")
  delete(@path customerIdOrKey: CustomerIdentifier): Customer | NotFoundError | CommonErrors;
}

/**
 * Query params for listing customers.
 */
@friendlyName("queryCustomerList")
model ListCustomersParams extends PaginatedQuery {
  /**
   * Include deleted customers.
   */
  @query
  includeDeleted?: boolean = false;
}

/**
 * A unique customer identifier.
 */
@extension("x-go-type", "string")
@friendlyName("CustomerIdentifier")
union CustomerIdentifier {
  ULID,
  Key,
}

/**
 * A customer object.
 */
@friendlyName("Customer")
@example(#{
  id: "01G65Z755AFWAKHE12NY0CQ9FH",
  name: "ACME Inc.",
  usageAttribution: #{ subjectKeys: #["my_subject_key"] },
  taxProvider: "stripe_tax",
  invoicingProvider: "stripe_invoicing",
  paymentProvider: "stripe_payments",
  external: #{ stripeCustomerId: "cus_xxxxxxxxxxxxxx" },
  createdAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
  updatedAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
})
model Customer {
  ...Resource;

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

  /**
   * The tax provider for the customer.
   */
  @summary("Tax Provider")
  taxProvider?: TaxProvider;

  /**
   * The invoicing provider for the customer.
   */
  @summary("Invoicing Provider")
  invoicingProvider?: InvoicingProvider;

  /**
   * The payment provider for the customer.
   */
  @summary("Payment Provider")
  paymentProvider?: PaymentProvider;

  /**
   * External mappings for the customer.
   */
  @summary("External Mappings")
  external?: CustomerExternalMapping;
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

/**
 * External mappings for the customer.
 */
@friendlyName("CustomerExternalMapping")
model CustomerExternalMapping {
  /**
   * The Stripe customer ID.
   * Mapping to a Stripe Customer object.
   * Required to use Stripe as an invocing provider.
   */
  @summary("Stripe Customer")
  stripeCustomerId?: string;
}
