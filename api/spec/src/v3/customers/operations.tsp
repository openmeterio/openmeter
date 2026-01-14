import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./customer.tsp";
import "./billing.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Customers;

/**
 * Filter options for listing customers.
 */
@friendlyName("ListCustomersParamsFilter")
model ListCustomersParamsFilter {
  /**
   * Filter customers by key.
   * Case-insensitive partial match.
   */
  key?: Shared.ExternalResourceKey;
}

interface CustomersOperations {
  @post
  @operationId("create-customer")
  @summary("Create customer")
  create(
    @body
    customer: Shared.CreateRequest<Customer>,
  ): Shared.CreateResponse<Customer> | Common.ErrorResponses;

  @get
  @operationId("get-customer")
  @summary("Get customer")
  get(
    @path customerId: Shared.ULID,
  ): Shared.GetResponse<Customer> | Common.NotFound | Common.ErrorResponses;

  @get
  @operationId("list-customers")
  @summary("List customers")
  list(
    ...Common.PagePaginationQuery,

    /**
     * Sort customers returned in the response.
     * Supported sort attributes are:
     * - `id`
     * - `name` (default)
     * - `created_at`
     *
     * The `asc` suffix is optional as the default sort order is ascending.
     * The `desc` suffix is used to specify a descending order.
     */
    @query(#{ name: "sort" })
    sort?: Common.SortQuery,

    /**
     * Filter customers returned in the response.
     *
     * To filter customers by key add the following query param: filter[key]=my-db-id
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListCustomersParamsFilter,
  ): Shared.PagePaginatedResponse<Customer> | Common.ErrorResponses;

  @put
  @operationId("upsert-customer")
  @summary("Upsert customer")
  upsert(
    @path customerId: Shared.ULID,

    @body
    customer: Shared.UpsertRequest<Customer>,
  ):
    | Shared.UpsertResponse<Customer>
    | Common.Gone
    | Common.NotFound
    | Common.ErrorResponses;

  @delete
  @operationId("delete-customer")
  @summary("Delete customer")
  delete(
    @path customerId: Shared.ULID,
  ): Shared.DeleteResponse | Common.NotFound | Common.ErrorResponses;
}

interface CustomerBillingOperations {
  @get
  @operationId("get-customer-billing")
  @summary("Get customer billing data")
  @extension(Shared.InternalExtension, true)
  @extension(Shared.UnstableExtension, true)
  get(
    @path customerId: Shared.ULID,
  ): Shared.GetResponse<CustomerBillingData> | Common.NotFound | Common.ErrorResponses;

  @put
  @operationId("update-customer-billing")
  @summary("Update customer billing data")
  @extension(Shared.InternalExtension, true)
  @extension(Shared.UnstableExtension, true)
  upsert(
    @path customerId: Shared.ULID,
    @body body: Shared.UpsertRequest<CustomerBillingData>,
  ): Shared.UpsertResponse<CustomerBillingData> | Common.NotFound | Common.ErrorResponses;

  @put
  @route("/app-data")
  @operationId("update-customer-billing-app-data")
  @summary("Update customer billing app data")
  @extension(Shared.InternalExtension, true)
  @extension(Shared.UnstableExtension, true)
  upsertAppData(
    @path customerId: Shared.ULID,
    @body body: Shared.UpsertRequest<Apps.AppCustomerData>,
  ): Shared.UpsertResponse<Apps.AppCustomerData> | Common.NotFound | Common.ErrorResponses;

  /**
   * Create a [Stripe Checkout Session](https://docs.stripe.com/payments/checkout) for the customer.
   *
   * Creates a Checkout Session for collecting payment method information from customers.
   * The session operates in "setup" mode, which collects payment details without charging
   * the customer immediately. The collected payment method can be used for future
   * subscription billing.
   *
   * For hosted checkout sessions, redirect customers to the returned URL. For embedded
   * sessions, use the client_secret to initialize Stripe.js in your application.
   */
  @post
  @route("/stripe/checkout-sessions")
  @operationId("create-customer-stripe-checkout-session")
  @summary("Create Stripe Checkout Session")
  @extension(Shared.InternalExtension, true)
  @extension(Shared.UnstableExtension, true)
  createCheckoutSession(
    @path customerId: Shared.ULID,
    @body body: CustomerBillingStripeCreateCheckoutSessionRequest,
  ): Shared.CreateResponse<Apps.CreateStripeCheckoutSessionResult> | Common.NotFound | Common.ErrorResponses;
}

/**
 * Request to create a Stripe Checkout Session for the customer.
 *
 * Checkout Sessions are used to collect payment method information from customers
 * in a secure, Stripe-hosted interface. This integration uses setup mode to collect
 * payment methods that can be charged later for subscription billing.
 */
@friendlyName("BillingCustomerStripeCreateCheckoutSessionRequest")
model CustomerBillingStripeCreateCheckoutSessionRequest {
  /**
   * The Stripe app ID to use for creating the checkout session.
   *
   * If not provided, the default Stripe app will be used if exists.
   */
  app_id?: Shared.ULID;

  /**
   * Existing Stripe customer ID to use for the checkout session.
   *
   * If not provided, a new Stripe customer will be created, or the billing
   * system will use the customer's existing default Stripe customer ID if available.
   */
  stripe_customer_id?: string;

  /**
   * Options for configuring the Stripe Checkout Session.
   *
   * These options are passed directly to Stripe's checkout session creation API.
   * See: https://docs.stripe.com/api/checkout/sessions/create
   */
  options: Apps.CreateStripeCheckoutSessionRequestOptions;
}
