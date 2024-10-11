import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";

import "./app.tsp";
import "./capability.tsp";
import "../errors.tsp";
import "../customer.tsp";
import "../pagination.tsp";
import "../types.tsp";

using OpenMeter.App;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.App;

/**
 * Stripe specific APIs.
 */
@tag("Stripe")
interface AppStripe {
  /**
   * Stripe webhook.
   */
  @post
  @route("/api/v1/apps/{id}/stripe/webhook")
  @operationId("appStripeWebhook")
  webhook(@path id: ULID, @body body: StripeWebhookEvent): StripeWebhookResponse | NotFoundError | CommonErrors;

  /**
   * Create checkout session.
   */
  @post
  @route("/api/v1/integration/stripe/checkout/sessions")
  @operationId("createStripeCheckoutSession")
  createCheckoutSession(@body body: CreateStripeCheckoutSessionRequest): {
    @statusCode _: 201;
    @body body: CreateStripeCheckoutSessionResponse;
  } | NotFoundError | CommonErrors;
}

/**
 * Stripe webhook response.
 */
@friendlyName("StripeWebhookResponse")
model StripeWebhookResponse {
  namespaceId: ULID;
  appId: ULID;
  customerId?: ULID;
}

/**
 * Create Stripe checkout session customer ID.
 */
@friendlyName("CustomerId")
model CustomerId {
  id: ULID;
}

/**
 * Create Stripe checkout session request.
 */
@friendlyName("CreateStripeCheckoutSessionRequest")
model CreateStripeCheckoutSessionRequest {
  /**
   * If not provided, the default Stripe app is used if any.
   */
  appId?: ULID;

  /**
   * Provide a customer ID to use an existing OpenMeter customer.
   * or provide a customer object to create a new customer.
   */
  customer: CustomerId | Customer;

  /**
   * Stripe customer ID.
   * If not provided OpenMeter creates a new Stripe customer or
   * uses the OpenMeter customer's default Stripe customer ID.
   */
  stripeCustomerId?: string;

  /**
   * Options passed to Stripe when creating the checkout session.
   */
  options: CreateStripeCheckoutSessionRequestOptions;
}

/**
 * Create Stripe checkout session options
 */
@friendlyName("CreateStripeCheckoutSessionRequestOptions")
model CreateStripeCheckoutSessionRequestOptions {
  currency?: CurrencyCode;
  cancelURL?: string;
  clientReferenceID?: string;
  customText?: CheckoutSessionCustomTextParams;
  metadata?: Record<string>;
  returnURL?: string;
  successURL?: string;
  uiMode?: CheckoutSessionUIMode;
  paymentMethodTypes?: string[];
}

/**
 * Create Stripe Checkout Session response.
 */
@friendlyName("CreateStripeCheckoutSessionResponse")
model CreateStripeCheckoutSessionResponse {
  /**
   * The OpenMeter customer ID.
   */
  customerId: ULID;

  /**
   * The Stripe customer ID.
   */
  stripeCustomerId: string;

  /**
   * The checkout session ID.
   */
  sessionId: string;

  /**
   * The checkout session setup intent ID.
   */
  setupIntentId: string;

  /**
   * URL to show the checkout session.
   */
  url: string;

  /**
   * Mode
   * Always `setup` for now.
   */
  mode: StripeCheckoutSessionMode;

  /**
   * Cancel URL.
   */
  cancelURL?: string;

  /**
   * Success URL.
   */
  successURL?: string;

  /**
   * Return URL.
   */
  returnURL?: string;
}

/**
 * Stripe CheckoutSession.mode
 * @see https://docs.stripe.com/api/checkout/sessions/object#checkout_session_object-mode
 *
 * Stripe checkout session mode.
 */
@friendlyName("StripeCheckoutSessionMode")
enum StripeCheckoutSessionMode {
  Setup: "setup",
}

/**
 * Stripe CheckoutSession.ui_mode
 * @see https://docs.stripe.com/api/checkout/sessions/object#checkout_session_object-ui_mode
 *
 * Stripe checkout session UI mode.
 */
@friendlyName("CheckoutSessionUIMode")
enum CheckoutSessionUIMode {
  Embedded: "embedded",
  Hosted: "hosted",
}

/**
 * Stripe CheckoutSession.custom_text
 * @see https://docs.stripe.com/api/checkout/sessions/object#checkout_session_object-custom_text
 *
 * Display additional text for your customers using custom text.
 */
@friendlyName("CheckoutSessionCustomTextAfterSubmitParams")
model CheckoutSessionCustomTextParams {
  // Custom text that should be displayed after the payment confirmation button.
  afterSubmit?: {
    // Text may be up to 1200 characters in length.
    message?: string;
  };

  // Custom text that should be displayed alongside shipping address collection.
  shippingAddress?: {
    // Text may be up to 1200 characters in length.
    message?: string;
  };

  // Custom text that should be displayed alongside the payment confirmation button.
  submit?: {
    // Text may be up to 1200 characters in length.
    message?: string;
  };

  // Custom text that should be displayed in place of the default terms of service agreement text.
  termsOfServiceAcceptance?: {
    // Text may be up to 1200 characters in length.
    message?: string;
  };
}

/**
 * A installed Stripe app object.
 */
@friendlyName("StripeApp")
@example(#{
  id: "01G65Z755AFWAKHE12NY0CQ9FH",
  type: Type.Stripe,
  name: "Stripe",
  status: AppStatus.Ready,
  listing: #{
    type: Type.Stripe,
    name: "Stripe",
    description: "Stripe interation allows you to collect payments with Stripe.",
    iconUrl: "/images/stripe.png",
    capabilities: #[
      #{
        type: CapabilityType.CalculateTax,
        key: "stripe_calculate_tax",
        name: "Calculate Tax",
        description: "Stripe Tax calculates tax portion of the invoices.",
        requirements: #[Requirements.CustomerExternalStripeCustomerId],
      },
      #{
        type: CapabilityType.InvoiceCustomers,
        key: "stripe_invoice_customers",
        name: "Invoice Customers",
        description: "Stripe invoices customers with due amount.",
        requirements: #[Requirements.CustomerExternalStripeCustomerId],
      },
      #{
        type: CapabilityType.CollectPayments,
        key: "stripe_collect_payments",
        name: "Collect Payments",
        description: "Stripe payments collects outstanding revenue with Stripe customer's default payment method.",
        requirements: #[Requirements.CustomerExternalStripeCustomerId],
      }
    ],
  },
  createdAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
  updatedAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),

  // Stripe specific fields
  stripeAccountId: "acct_123456789",

  livemode: true,
})
model StripeApp {
  ...AppBase;

  /**
   * The app's type is Stripe.
   */
  type: Type.Stripe;

  /**
   * The Stripe account ID.
   */
  stripeAccountId: string;

  /**
   * Livemode, true if the app is in production mode.
   */
  livemode: boolean;
}

/**
 * Stripe webhook event.
 */
@friendlyName("StripeWebhookEvent")
model StripeWebhookEvent {
  /**
   * The event ID.
   */
  id: string;

  /**
   * The event type.
   */
  type: string;

  /**
   * Live mode.
   */
  livemode: boolean;

  /**
   * The event created timestamp.
   */
  created: int32;

  /**
   * The event data.
   */
  data: {
    object: StripeSetupIntent;
  };
}

/**
 * Stripe setup intent.
 */
@friendlyName("StripeSetupIntent")
model StripeSetupIntent {
  /**
   * The setup intent id.
   */
  id: string;

  /**
   * The setup intent status.
   */
  status: StripePaymentIntentStatus;

  /**
   * The setup intent payment method.
   */
  payment_method?: string;

  /**
   * The setup intent payment method types.
   */
  payment_method_types?: string[];

  /**
   * The setup intent customer.
   */
  customer: string;

  /**
   * The setup intent metadata.
   */
  metadata: Record<string>;
}

/**
 * Stripe payment intent status.
 */
enum StripePaymentIntentStatus {
  Canceled: "canceled",
  Processing: "processing",
  RequiresAction: "requires_action",
  RequiresConfirmation: "requires_confirmation",
  RequiresPaymentMethod: "requires_payment_method",
  Succeeded: "succeeded",
}
