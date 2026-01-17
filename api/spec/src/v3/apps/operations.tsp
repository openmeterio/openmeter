import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./app.tsp";
import "./custominvoicing.tsp";
import "./stripe.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Apps;

/**
 * App install request.
 */
@friendlyName("BillingAppInstallRequest")
model InstallAppRequest {
  /**
   * The name of the app.
   *
   * If name is not provided defaults to the catalog item's name.
   */
  name?: string;

  /**
   * If true, a billing profile will be created for the app.
   *
   * The Stripe app will be also set as the default billing profile if the current default is a Sandbox app.
   */
  create_billing_profile?: boolean = true;
}

/**
 * App API key install request.
 */
@friendlyName("BillingAppAPIKeyInstallRequest")
model InstallAppWithAPIKeyRequest {
  ...InstallAppRequest;

  /**
   * The API key for the provider.
   *
   * For example, the Stripe API key.
   */
  api_key: string;
}

interface AppCatalogOperations {
  @get
  @operationId("list-app-catalog-items")
  @summary("List available apps in the app catalog")
  @extension(Shared.InternalExtension, true)
  @extension(Shared.UnstableExtension, true)
  list(
    ...Common.PagePaginationQuery,
  ): Shared.PagePaginatedResponse<AppCatalogItem> | Common.ErrorResponses;

  @get
  @route("/{type}")
  @operationId("get-app-catalog-item")
  @summary("Get app catalog item by type")
  @extension(Shared.InternalExtension, true)
  @extension(Shared.UnstableExtension, true)
  get(
    @path type: AppType,
  ): Shared.GetResponse<AppCatalogItem> | Common.NotFound | Common.ErrorResponses;

  /**
   * Install an app from the catalog via API Key.
   */
  @post
  @route("/{type}/install/api-key")
  @operationId("install-app-via-api-key")
  @summary("Install app via API key")
  @extension(Shared.InternalExtension, true)
  @extension(Shared.UnstableExtension, true)
  installWithAPIKey(
    @path type: AppType,
    @body body: InstallAppWithAPIKeyRequest,
  ): Shared.CreateResponse<App> | Common.NotFound | Common.ErrorResponses;

  /**
   * Install an app from the catalog.
   */
  @post
  @route("/{type}/install")
  @operationId("install-app")
  @summary("Install app")
  @extension(Shared.InternalExtension, true)
  @extension(Shared.UnstableExtension, true)
  install(
    @path type: AppType,
    @body body: InstallAppRequest,
  ): Shared.CreateResponse<App> | Common.NotFound | Common.ErrorResponses;
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
  @extension(Shared.InternalExtension, true)
  @extension(Shared.UnstableExtension, true)
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
  @extension(Shared.InternalExtension, true)
  @extension(Shared.UnstableExtension, true)
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
  @extension(Shared.InternalExtension, true)
  @extension(Shared.UnstableExtension, true)
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
  @extension(Shared.InternalExtension, true)
  @extension(Shared.UnstableExtension, true)
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
  @extension(Shared.InternalExtension, true)
  @extension(Shared.UnstableExtension, true)
  createCheckoutSession(
    /**
     * Checkout session configuration including customer and options.
     */
    @body body: CreateStripeCheckoutSessionRequest,
  ): Shared.CreateResponse<CreateStripeCheckoutSessionResult> | Common.ErrorResponses;
}
