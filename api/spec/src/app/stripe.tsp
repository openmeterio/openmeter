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
@tag("Apps (Private Preview)")
interface AppStripe {
  /**
   * Handle stripe webhooks for apps.
   */
  @post
  @route("/api/v1/apps/{id}/stripe/webhook")
  @operationId("appStripeWebhook")
  @useAuth(NoAuth)
  @summary("Stripe webhook")
  webhook(
    @path id: ULID,
    @body body: StripeWebhookEvent,
  ): StripeWebhookResponse | NotFoundError | CommonErrors;

  /**
   * Update the Stripe API key.
   */
  @put
  @route("/api/v1/apps/{id}/stripe/api-key")
  @operationId("updateStripeAPIKey")
  @summary("Update Stripe API key")
  updateStripeAPIKey(
    @path id: ULID,
    @body request: StripeAPIKeyInput,
  ): void | OpenMeter.CommonErrors;

  /**
   * Create checkout session.
   */
  @post
  @route("/api/v1/stripe/checkout/sessions")
  @operationId("createStripeCheckoutSession")
  @summary("Create checkout session")
  createCheckoutSession(@body body: CreateStripeCheckoutSessionRequest): {
    @statusCode _: 201;
    @body body: CreateStripeCheckoutSessionResult;
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
  customer: CustomerId | Rest.Resource.ResourceCreateModel<Customer>;

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
 * See https://docs.stripe.com/api/checkout/sessions/create
 */
@friendlyName("CreateStripeCheckoutSessionRequestOptions")
model CreateStripeCheckoutSessionRequestOptions {
  // Settings for automatic tax lookup for this session and resulting payments, invoices, and subscriptions.
  automaticTax?: CreateStripeCheckoutSessionAutomaticTax;

  // Specify whether Checkout should collect the customer’s billing address. Defaults to auto.
  billingAddressCollection?: CreateStripeCheckoutSessionBillingAddressCollection;

  // If set, Checkout displays a back button and customers will be directed to this URL if they decide to cancel payment and return to your website.
  // This parameter is not allowed if ui_mode is embedded.
  cancelURL?: string;

  // A unique string to reference the Checkout Session. This can be a customer ID, a cart ID, or similar, and can be used to reconcile the session with your internal systems.
  clientReferenceID?: string;

  // Configure fields for the Checkout Session to gather active consent from customers.
  consentCollection?: CreateStripeCheckoutSessionConsentCollection;

  // Three-letter ISO currency code, in lowercase.
  currency?: CurrencyCode;

  // Display additional text for your customers using custom text.
  customText?: CheckoutSessionCustomTextParams;

  // The Epoch time in seconds at which the Checkout Session will expire.
  // It can be anywhere from 30 minutes to 24 hours after Checkout Session creation. By default, this value is 24 hours from creation.
  expiresAt?: integer;

  // The IETF language tag of the locale Checkout is displayed in. If blank or auto, the browser’s locale is used.
  locale?: string;

  // Set of key-value pairs that you can attach to an object.
  // This can be useful for storing additional information about the object in a structured format.
  // Individual keys can be unset by posting an empty value to them.
  // All keys can be unset by posting an empty value to metadata.
  metadata?: Record<string>;

  // The URL to redirect your customer back to after they authenticate or cancel their payment on the payment method’s app or site.
  // This parameter is required if ui_mode is embedded and redirect-based payment methods are enabled on the session.
  returnURL?: string;

  // The URL to which Stripe should send customers when payment or setup is complete.
  // This parameter is not allowed if ui_mode is embedded.
  // If you’d like to use information from the successful Checkout Session on your page, read the guide on customizing your success page:
  // https://docs.stripe.com/payments/checkout/custom-success-page
  successURL?: string;

  // The UI mode of the Session. Defaults to hosted.
  uiMode?: CheckoutSessionUIMode;

  // A list of the types of payment methods (e.g., card) this Checkout Session can accept.
  paymentMethodTypes?: string[];

  // This parameter applies to ui_mode: embedded. Defaults to always.
  // Learn more about the redirect behavior of embedded sessions at
  // https://docs.stripe.com/payments/checkout/custom-success-page?payment-ui=embedded-form
  redirectOnCompletion: CreateStripeCheckoutSessionRedirectOnCompletion;

  // Controls tax ID collection during checkout.
  taxIdCollection?: CreateCheckoutSessionTaxIdCollection;

  // TODO: Fields we haven't implemented yet.
  // Please open an issue or pull request if you need them:
  // https://github.com/openmeterio/openmeter/issues
  // customFields: unknown;
  // paymentIntentData: unknown;
  // paymentMethodConfiguration: unknown;
  // paymentMethodData: unknown;
  // paymentMethodOptions: unknown;
  // setupIntentData: unknown;
}

/**
 * Settings for automatic tax lookup for this session and resulting payments and invoices.
 */
@friendlyName("CreateStripeCheckoutSessionAutomaticTax")
model CreateStripeCheckoutSessionAutomaticTax {
  // Set to true to calculate tax automatically using the customer’s location.
  // Enabling this parameter causes Checkout to collect any billing address information necessary for tax calculation.
  enabled: boolean;
}

/**
 * Specify whether Checkout should collect the customer’s billing address.
 */
@friendlyName("CreateStripeCheckoutSessionBillingAddressCollection")
enum CreateStripeCheckoutSessionBillingAddressCollection {
  // Checkout will only collect the billing address when necessary.
  // When using automatic_tax, Checkout will collect the minimum number of fields required for tax calculation.
  Auto: "auto",

  // Checkout will always collect the customer’s billing address.
  Required: "required",
}

/**
 * Configure fields for the Checkout Session to gather active consent from customers.
 */
@friendlyName("CreateStripeCheckoutSessionConsentCollection")
model CreateStripeCheckoutSessionConsentCollection {
  // Determines the position and visibility of the payment method reuse agreement in the UI.
  // When set to auto, Stripe’s defaults will be used. When set to hidden, the payment method reuse agreement text will always be hidden in the UI.
  paymentMethodReuseAgreement?: CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreement;

  // If set to auto, enables the collection of customer consent for promotional communications.
  // The Checkout Session will determine whether to display an option to opt into promotional
  // communication from the merchant depending on the customer’s locale. Only available to US merchants.
  promotions?: CreateStripeCheckoutSessionConsentCollectionPromotions;

  // If set to required, it requires customers to check a terms of service checkbox before being able to pay.
  // There must be a valid terms of service URL set in your Stripe Dashboard settings.
  // https://dashboard.stripe.com/settings/public
  termsOfService?: CreateStripeCheckoutSessionConsentCollectionTermsOfService;
}

/**
 * Create Stripe checkout session payment method reuse agreement.
 */
@friendlyName("CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreement")
model CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreement {
  position?: CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition;
}

/**
 * Create Stripe checkout session consent collection agreement position.
 */
@friendlyName("CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition")
enum CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition {
  // Uses Stripe defaults to determine the visibility and position of the payment method reuse agreement.
  Auto: "auto",

  // Hides the payment method reuse agreement.
  Hidden: "hidden",
}

/**
 * Create Stripe checkout session consent collection promotions.
 */
@friendlyName("CreateStripeCheckoutSessionConsentCollectionPromotions")
enum CreateStripeCheckoutSessionConsentCollectionPromotions {
  // Enable the collection of customer consent for promotional communications.
  // The Checkout Session will determine whether to display an option to opt into promotional communication from the merchant depending on if a customer is provided,
  // and if that customer has consented to receiving promotional communications from the merchant in the past.
  Auto: "auto",

  // Checkout will not collect customer consent for promotional communications.
  None: "none",
}

/**
 * Create Stripe checkout session consent collection terms of service.
 */
@friendlyName("CreateStripeCheckoutSessionConsentCollectionTermsOfService")
enum CreateStripeCheckoutSessionConsentCollectionTermsOfService {
  // Does not display checkbox for the terms of service agreement.
  None: "none",

  // Displays a checkbox for the terms of service agreement which requires customer to check before being able to pay.
  Required: "required",
}

/**
 * Create Stripe checkout session redirect on completion.
 */
@friendlyName("CreateStripeCheckoutSessionRedirectOnCompletion")
enum CreateStripeCheckoutSessionRedirectOnCompletion {
  // The Session will always redirect to the return_url after successful confirmation.
  Always: "always",

  // The Session will only redirect to the return_url after a redirect-based payment method is used.
  IfRequired: "if_required",

  // The Session will never redirect to the return_url, and redirect-based payment methods will be disabled.
  Never: "never",
}

/**
 * Create Stripe checkout session tax ID collection.
 */
@friendlyName("CreateCheckoutSessionTaxIdCollection")
model CreateCheckoutSessionTaxIdCollection {
  // Enable tax ID collection during checkout. Defaults to false.
  enabled: boolean;

  // Describes whether a tax ID is required during checkout. Defaults to never.
  required?: CreateCheckoutSessionTaxIdCollectionRequired;
}

/**
 * Create Stripe checkout session tax ID collection required.
 */
@friendlyName("CreateCheckoutSessionTaxIdCollectionRequired")
enum CreateCheckoutSessionTaxIdCollectionRequired {
  // A tax ID will be required if collection is supported for the selected billing address country.
  // See: https://docs.stripe.com/tax/checkout/tax-ids#supported-types
  IfSupported: "if_supported",

  // Tax ID collection is never required.
  Never: "never",
}

/**
 * Create Stripe Checkout Session response.
 */
@friendlyName("CreateStripeCheckoutSessionResult")
model CreateStripeCheckoutSessionResult {
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
  url?: string;

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
  default: true,
  listing: #{
    type: Type.Stripe,
    name: "Stripe",
    description: "Stripe interation allows you to collect payments with Stripe.",
    capabilities: #[
      #{
        type: CapabilityType.CalculateTax,
        key: "stripe_calculate_tax",
        name: "Calculate Tax",
        description: "Stripe Tax calculates tax portion of the invoices.",
      },
      #{
        type: CapabilityType.InvoiceCustomers,
        key: "stripe_invoice_customers",
        name: "Invoice Customers",
        description: "Stripe invoices customers with due amount.",
      },
      #{
        type: CapabilityType.CollectPayments,
        key: "stripe_collect_payments",
        name: "Collect Payments",
        description: "Stripe payments collects outstanding revenue with Stripe customer's default payment method.",
      }
    ],
  },
  createdAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
  updatedAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),

  // Stripe specific fields
  stripeAccountId: "acct_123456789",

  livemode: true,
  maskedAPIKey: "sk_live_************abc",
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

  /**
   * The masked API key.
   * Only shows the first 8 and last 3 characters.
   */
  maskedAPIKey: string;
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
@friendlyName("StripePaymentIntentStatus")
enum StripePaymentIntentStatus {
  Canceled: "canceled",
  Processing: "processing",
  RequiresAction: "requires_action",
  RequiresConfirmation: "requires_confirmation",
  RequiresPaymentMethod: "requires_payment_method",
  Succeeded: "succeeded",
}

/**
 * The Stripe API key input.
 * Used to authenticate with the Stripe API.
 */
@friendlyName("StripeAPIKeyInput")
model StripeAPIKeyInput {
  secretAPIKey: string;
}
