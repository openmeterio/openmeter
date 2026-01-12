import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./app.tsp";
import "./oauth.tsp";
import "./custominvoicing.tsp";
import "./stripe.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Apps;

interface AppCatalogOperations {
  @get
  @operationId("list-app-catalog-items")
  @summary("List available apps in the app catalog")
  list(
    ...Common.PagePaginationQuery,
  ): Shared.PagePaginatedResponse<AppCatalogItem> | Common.ErrorResponses;

  @get
  @route("/{type}")
  @operationId("get-app-catalog-item")
  @summary("Get app catalog item by type")
  get(
    @path type: AppType,
  ): AppCatalogItem | Common.NotFound | Common.ErrorResponses;

  @get
  @route("/{type}/install/oauth2")
  @operationId("get-app-catalog-item-oauth2-install-url")
  @summary("Get OAuth2 install URL for an app catalog item")
  getOAuth2InstallURL(
    @path type: AppType,
  ): Shared.GetResponse<AppCatalogItem> | Common.NotFound | Common.ErrorResponses;
}

/**
 * Custom Invoicing app operations for synchronizing invoice state and payment status.
 */
interface AppCustomInvoicingOperations {
  /**
   * Submit draft synchronization results.
   *
   * This endpoint is called by the integration after the invoice draft has been synchronized
   * with the external invoicing system. It allows setting invoice numbers, external IDs,
   * and mapping line items to external system entities.
   */
  @post
  @route("/{invoiceId}/draft/synchronized")
  @operationId("submit-custom-invoicing-draft-synchronized")
  @summary("Submit draft synchronization results")
  draftSynchronized(
    /**
     * The invoice ID.
     */
    @path invoiceId: Shared.ULID,

    /**
     * The synchronization result.
     */
    @body body: CustomInvoicingDraftSynchronizedRequest,
  ): void | Common.ErrorResponses;

  /**
   * Submit issuing synchronization results.
   *
   * This endpoint is called by the integration after the invoice has been issued/finalized
   * in the external invoicing system. It allows finalizing invoice and payment details,
   * including invoice numbers and timestamps for when the invoice was sent to the customer.
   */
  @post
  @route("/{invoiceId}/issuing/synchronized")
  @operationId("submit-custom-invoicing-issuing-synchronized")
  @summary("Submit issuing synchronization results")
  issuingSynchronized(
    /**
     * The invoice ID.
     */
    @path invoiceId: Shared.ULID,

    /**
     * The finalization result.
     */
    @body body: CustomInvoicingFinalizedRequest,
  ): void | Common.ErrorResponses;

  /**
   * Update payment status.
   *
   * This endpoint allows the integration to update the payment status of an invoice
   * based on events from the external payment system. It supports various payment
   * lifecycle events such as paid, failed, overdue, and void.
   */
  @post
  @route("/{invoiceId}/payment/status")
  @operationId("update-custom-invoicing-payment-status")
  @summary("Update payment status")
  updatePaymentStatus(
    /**
     * The invoice ID.
     */
    @path invoiceId: Shared.ULID,

    /**
     * The payment status update.
     */
    @body body: CustomInvoicingUpdatePaymentStatusRequest,
  ): void | Common.ErrorResponses;
}

/**
 * Stripe app operations for payment collection and checkout sessions.
 */
interface AppStripeOperations {
  /**
   * Handle Stripe webhook events.
   *
   * This endpoint receives and processes webhook events from Stripe, such as
   * payment confirmations, subscription changes, and invoice updates. The endpoint
   * validates the webhook signature and processes events asynchronously.
   *
   * Webhooks must be configured in your Stripe Dashboard to point to this endpoint.
   * See: https://docs.stripe.com/webhooks
   */
  @post
  @route("/{appId}/webhook")
  @operationId("handle-stripe-webhook")
  @summary("Handle Stripe webhook")
  handleWebhook(
    /**
     * The Stripe app ID that should process this webhook.
     */
    @path appId: Shared.ULID,

    /**
     * The webhook event payload from Stripe.
     */
    @body body: StripeWebhookEvent,
  ): StripeWebhookResponse | Common.ErrorResponses;

  /**
   * Create a Stripe Checkout Session.
   *
   * Creates a Checkout Session for collecting payment method information from customers.
   * The session operates in "setup" mode, which collects payment details without charging
   * the customer immediately. The collected payment method can be used for future
   * subscription billing.
   *
   * For hosted checkout sessions, redirect customers to the returned URL. For embedded
   * sessions, use the client_secret to initialize Stripe.js in your application.
   *
   * See: https://docs.stripe.com/payments/checkout
   */
  @post
  @route("/checkout-sessions")
  @operationId("create-stripe-checkout-session")
  @summary("Create Stripe Checkout Session")
  createCheckoutSession(
    /**
     * Checkout session configuration including customer and options.
     */
    @body body: CreateStripeCheckoutSessionRequest,
  ): {
    @statusCode _: 201;
    @body body: CreateStripeCheckoutSessionResult;
  } | Common.ErrorResponses;
}
