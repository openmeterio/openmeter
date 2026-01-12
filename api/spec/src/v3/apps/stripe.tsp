import "../shared/index.tsp";
import "./app.tsp";
import "../customers/customer.tsp";

namespace Apps;

/**
 * Stripe app.
 */
@friendlyName("BillingAppStripe")
model AppStripe {
  ...AppBase<AppType.Stripe>;

  /**
   * The Stripe account ID associated with the connected Stripe account.
   */
  @visibility(Lifecycle.Read)
  account_id: string;

  /**
   * Indicates whether the app is connected to a live Stripe account.
   */
  @visibility(Lifecycle.Read)
  livemode: boolean;

  /**
   * The masked Stripe API key that only exposes the first and last few characters.
   */
  @visibility(Lifecycle.Read)
  masked_api_key: string;

  /**
   * The Stripe secret API key used to authenticate API requests.
   */
  @visibility(Lifecycle.Create, Lifecycle.Update)
  @secret
  secret_api_key?: string;
}

/**
 * Stripe webhook event received from Stripe's webhook system.
 *
 * Stripe sends webhook events to notify the billing system about asynchronous events.
 */
@friendlyName("BillingAppStripeWebhookEvent")
model StripeWebhookEvent {
  /**
   * Unique identifier for the event.
   */
  id: string;

  /**
   * The type of event (e.g., "payment_intent.succeeded", "invoice.paid").
   *
   * See Stripe's documentation for the full list of event types:
   * https://docs.stripe.com/api/events/types
   */
  type: string;

  /**
   * Indicates whether the event occurred in live mode or test mode.
   */
  livemode: boolean;

  /**
   * Unix timestamp when the event was created.
   */
  created: int32;

  /**
   * Object containing the event's data payload.
   *
   * The structure varies by event type and is validated using the
   * Stripe SDK which handles API versioning.
   */
  data: {
    object: unknown;
  };
}

/**
 * Response returned after processing a Stripe webhook.
 */
@friendlyName("BillingAppStripeWebhookResponse")
model StripeWebhookResponse {
  /**
   * The namespace ID where the event was processed.
   */
  namespace_id: Shared.ULID;

  /**
   * The Stripe app ID that handled the webhook.
   */
  app_id: Shared.ULID;

  /**
   * The customer ID if the event was related to a specific customer.
   */
  customer_id?: Shared.ULID;

  /**
   * Optional message providing additional context about the webhook processing.
   */
  message?: string;
}

/**
 * Request to create a Stripe Checkout Session.
 *
 * Checkout Sessions are used to collect payment method information from customers
 * in a secure, Stripe-hosted interface. This integration uses setup mode to collect
 * payment methods that can be charged later for subscription billing.
 */
@friendlyName("BillingAppStripeCreateCheckoutSessionRequest")
model CreateStripeCheckoutSessionRequest {
  /**
   * The Stripe app ID to use for creating the checkout session.
   *
   * If not provided, the default Stripe app will be used if exists.
   */
  app_id?: Shared.ULID;

  /**
   * Customer identifier for creating or associating the checkout session.
   *
   * Can be an existing customer ID, customer key, or customer creation data.
   */
  customer: StripeCustomerId | StripeCustomerKey | StripeCustomerCreate;

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
  options: CreateStripeCheckoutSessionRequestOptions;
}

/**
 * Reference to an existing customer by ID.
 */
@friendlyName("BillingAppStripeCustomerId")
model StripeCustomerId {
  /**
   * The customer's unique identifier (ULID).
   */
  id: Shared.ULID;
}

/**
 * Reference to an existing customer by key.
 */
@friendlyName("BillingAppStripeCustomerKey")
model StripeCustomerKey {
  /**
   * The customer's unique key.
   */
  key: Shared.ExternalResourceKey;
}

/**
 * Customer creation data for new customers during checkout session creation.
 */
@friendlyName("BillingAppStripeCustomerCreate")
model StripeCustomerCreate {
  ...OmitProperties<Customers.Customer, "currency">;

  /**
   * The customer's primary currency for billing.
   */
  currency: Shared.CurrencyCode;
}

/**
 * Configuration options for creating a Stripe Checkout Session.
 *
 * Based on Stripe's Checkout Session API parameters.
 * See: https://docs.stripe.com/api/checkout/sessions/create
 */
@friendlyName("BillingAppStripeCreateCheckoutSessionRequestOptions")
model CreateStripeCheckoutSessionRequestOptions {
  /**
   * Whether to collect the customer's billing address.
   *
   * Defaults to auto, which only collects the address when necessary for tax calculation.
   */
  billing_address_collection?: CreateStripeCheckoutSessionBillingAddressCollection = CreateStripeCheckoutSessionBillingAddressCollection.Auto;

  /**
   * URL to redirect customers who cancel the checkout session.
   *
   * Not allowed when ui_mode is "embedded".
   */
  cancel_url?: string;

  /**
   * Unique reference string for reconciling sessions with internal systems.
   *
   * Can be a customer ID, cart ID, or any other identifier.
   */
  client_reference_id?: string;

  /**
   * Controls which customer fields can be updated by the checkout session.
   */
  customer_update?: CreateStripeCheckoutSessionCustomerUpdate;

  /**
   * Configuration for collecting customer consent during checkout.
   */
  consent_collection?: CreateStripeCheckoutSessionConsentCollection;

  /**
   * Three-letter ISO 4217 currency code in lowercase.
   *
   * Required for payment mode sessions. Optional for setup mode sessions.
   */
  currency?: Shared.CurrencyCode;

  /**
   * Custom text to display during checkout at various stages.
   */
  custom_text?: CheckoutSessionCustomTextParams;

  /**
   * Unix timestamp when the checkout session expires.
   *
   * Can be 30 minutes to 24 hours from creation. Defaults to 24 hours.
   */
  expires_at?: int64;

  /**
   * IETF language tag for the checkout UI locale.
   *
   * If blank or "auto", uses the browser's locale. Example: "en", "fr", "de".
   */
  locale?: string;

  /**
   * Set of key-value pairs to attach to the checkout session.
   *
   * Useful for storing additional structured information.
   */
  metadata?: Record<string>;

  /**
   * Return URL for embedded checkout sessions after payment authentication.
   *
   * Required if ui_mode is "embedded" and redirect-based payment methods are enabled.
   */
  return_url?: string;

  /**
   * Success URL to redirect customers after completing payment or setup.
   *
   * Not allowed when ui_mode is "embedded".
   * See: https://docs.stripe.com/payments/checkout/custom-success-page
   */
  success_url?: string;

  /**
   * The UI mode for the checkout session.
   *
   * "hosted" displays a Stripe-hosted page. "embedded" integrates directly into your app.
   * Defaults to "hosted".
   */
  ui_mode?: CheckoutSessionUIMode = CheckoutSessionUIMode.Hosted;

  /**
   * List of payment method types to enable (e.g., "card", "us_bank_account").
   *
   * If not specified, Stripe enables all relevant payment methods.
   */
  payment_method_types?: string[];

  /**
   * Redirect behavior for embedded checkout sessions.
   *
   * Controls when to redirect users after completion.
   * See: https://docs.stripe.com/payments/checkout/custom-success-page?payment-ui=embedded-form
   */
  redirect_on_completion?: CreateStripeCheckoutSessionRedirectOnCompletion;

  /**
   * Configuration for collecting tax IDs during checkout.
   */
  tax_id_collection?: CreateCheckoutSessionTaxIdCollection;
}

/**
 * Controls whether Checkout collects the customer's billing address.
 */
@friendlyName("BillingAppStripeCreateCheckoutSessionBillingAddressCollection")
enum CreateStripeCheckoutSessionBillingAddressCollection {
  /**
   * Collect billing address only when necessary (e.g., for tax calculation).
   */
  Auto: "auto",

  /**
   * Always collect the customer's billing address.
   */
  Required: "required",
}

/**
 * Checkout Session consent collection configuration.
 */
@friendlyName("BillingAppStripeCreateCheckoutSessionConsentCollection")
model CreateStripeCheckoutSessionConsentCollection {
  /**
   * Controls the visibility of payment method reuse agreement.
   */
  payment_method_reuse_agreement?: CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreement;

  /**
   * Enables collection of promotional communication consent.
   *
   * Only available to US merchants. When set to "auto", Checkout determines
   * whether to show the option based on the customer's locale.
   */
  promotions?: CreateStripeCheckoutSessionConsentCollectionPromotions;

  /**
   * Requires customers to accept terms of service before payment.
   *
   * Requires a valid terms of service URL in your Stripe Dashboard settings.
   */
  terms_of_service?: CreateStripeCheckoutSessionConsentCollectionTermsOfService;
}

/**
 * Controls which customer fields can be updated by the checkout session.
 */
@friendlyName("BillingAppStripeCreateCheckoutSessionCustomerUpdate")
model CreateStripeCheckoutSessionCustomerUpdate {
  /**
   * Whether to save the billing address to customer.address.
   *
   * Defaults to "never".
   */
  address?: CreateStripeCheckoutSessionCustomerUpdateBehavior = CreateStripeCheckoutSessionCustomerUpdateBehavior.Never;

  /**
   * Whether to save the customer name to customer.name.
   *
   * Defaults to "never".
   */
  name?: CreateStripeCheckoutSessionCustomerUpdateBehavior = CreateStripeCheckoutSessionCustomerUpdateBehavior.Never;

  /**
   * Whether to save shipping information to customer.shipping.
   *
   * Defaults to "never".
   */
  shipping?: CreateStripeCheckoutSessionCustomerUpdateBehavior = CreateStripeCheckoutSessionCustomerUpdateBehavior.Never;
}

/**
 * Behavior for updating customer fields from checkout session.
 */
@friendlyName("BillingAppStripeCreateCheckoutSessionCustomerUpdateBehavior")
enum CreateStripeCheckoutSessionCustomerUpdateBehavior {
  /**
   * Automatically determine whether to update the customer using session details.
   */
  Auto: "auto",

  /**
   * Never update the customer object.
   */
  Never: "never",
}

/**
 * Payment method reuse agreement configuration.
 */
@friendlyName("BillingAppStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreement")
model CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreement {
  /**
   * Position and visibility of the payment method reuse agreement.
   */
  position?: CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition;
}

/**
 * Position of payment method reuse agreement in the UI.
 */
@friendlyName("BillingAppStripeCreateCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition")
enum CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition {
  /**
   * Use Stripe defaults for visibility and position.
   */
  Auto: "auto",

  /**
   * Hide the payment method reuse agreement.
   */
  Hidden: "hidden",
}

/**
 * Promotional communication consent collection setting.
 */
@friendlyName("BillingAppStripeCreateCheckoutSessionConsentCollectionPromotions")
enum CreateStripeCheckoutSessionConsentCollectionPromotions {
  /**
   * Show promotional consent option based on customer context and locale.
   */
  Auto: "auto",

  /**
   * Do not collect promotional communication consent.
   */
  None: "none",
}

/**
 * Terms of service acceptance requirement.
 */
@friendlyName("BillingAppStripeCreateCheckoutSessionConsentCollectionTermsOfService")
enum CreateStripeCheckoutSessionConsentCollectionTermsOfService {
  /**
   * Do not display terms of service checkbox.
   */
  None: "none",

  /**
   * Require customers to accept terms of service before payment.
   */
  Required: "required",
}

/**
 * Redirect behavior for embedded checkout sessions.
 */
@friendlyName("BillingAppStripeCreateCheckoutSessionRedirectOnCompletion")
enum CreateStripeCheckoutSessionRedirectOnCompletion {
  /**
   * Always redirect to return_url after successful confirmation.
   */
  Always: "always",

  /**
   * Redirect only when a redirect-based payment method is used.
   */
  IfRequired: "if_required",

  /**
   * Never redirect, and disable redirect-based payment methods.
   */
  Never: "never",
}

/**
 * Tax ID collection configuration for checkout sessions.
 */
@friendlyName("BillingAppStripeCreateCheckoutSessionTaxIdCollection")
model CreateCheckoutSessionTaxIdCollection {
  /**
   * Enable tax ID collection during checkout.
   *
   * Defaults to false.
   */
  enabled: boolean;

  /**
   * Whether tax ID collection is required.
   *
   * Defaults to "never".
   */
  required?: CreateCheckoutSessionTaxIdCollectionRequired;
}

/**
 * Tax ID collection requirement level.
 */
@friendlyName("BillingAppStripeCreateCheckoutSessionTaxIdCollectionRequired")
enum CreateCheckoutSessionTaxIdCollectionRequired {
  /**
   * Require tax ID if collection is supported for the billing address country.
   *
   * See: https://docs.stripe.com/tax/checkout/tax-ids#supported-types
   */
  IfSupported: "if_supported",

  /**
   * Tax ID collection is never required.
   */
  Never: "never",
}

/**
 * Checkout Session UI mode.
 */
@friendlyName("BillingAppStripeCheckoutSessionUIMode")
enum CheckoutSessionUIMode {
  /**
   * Checkout UI embedded directly in your application.
   */
  Embedded: "embedded",

  /**
   * Checkout UI hosted on a Stripe-provided page.
   */
  Hosted: "hosted",
}

/**
 * Custom text displayed at various stages of the checkout flow.
 */
@friendlyName("BillingAppStripeCheckoutSessionCustomTextParams")
model CheckoutSessionCustomTextParams {
  /**
   * Text displayed after the payment confirmation button.
   */
  after_submit?: {
    /**
     * The custom message text (max 1200 characters).
     */
    @maxLength(1200)
    message?: string;
  };

  /**
   * Text displayed alongside shipping address collection.
   */
  shipping_address?: {
    /**
     * The custom message text (max 1200 characters).
     */
    @maxLength(1200)
    message?: string;
  };

  /**
   * Text displayed alongside the payment confirmation button.
   */
  submit?: {
    /**
     * The custom message text (max 1200 characters).
     */
    @maxLength(1200)
    message?: string;
  };

  /**
   * Text replacing the default terms of service agreement text.
   */
  terms_of_service_acceptance?: {
    /**
     * The custom message text (max 1200 characters).
     */
    @maxLength(1200)
    message?: string;
  };
}

/**
 * Result of creating a Stripe Checkout Session.
 *
 * Contains all the information needed to redirect customers to the checkout
 * or initialize an embedded checkout flow.
 */
#suppress "@openmeter/api-spec/repeated-prefix-grouping" "Field names match Stripe API response structure"
@friendlyName("BillingAppStripeCreateCheckoutSessionResult")
model CreateStripeCheckoutSessionResult {
  /**
   * The customer ID in the billing system.
   */
  customer_id: Shared.ULID;

  /**
   * The Stripe customer ID.
   */
  stripe_customer_id: string;

  /**
   * The Stripe checkout session ID.
   */
  session_id: string;

  /**
   * The setup intent ID created for collecting the payment method.
   */
  setup_intent_id: string;

  /**
   * Client secret for initializing Stripe.js on the client side.
   *
   * Required for embedded checkout sessions.
   * See: https://docs.stripe.com/payments/checkout/custom-success-page
   */
  client_secret?: string;

  /**
   * The client reference ID provided in the request.
   *
   * Useful for reconciling the session with your internal systems.
   */
  client_reference_id?: string;

  /**
   * Customer's email address if provided to Stripe.
   */
  customer_email?: string;

  /**
   * Currency code for the checkout session.
   */
  currency?: Shared.CurrencyCode;

  /**
   * Timestamp when the checkout session was created.
   */
  created_at: Shared.DateTime;

  /**
   * Timestamp when the checkout session will expire.
   */
  expires_at?: Shared.DateTime;

  /**
   * Metadata attached to the checkout session.
   */
  metadata?: Record<string>;

  /**
   * The status of the checkout session.
   *
   * See: https://docs.stripe.com/api/checkout/sessions/object#checkout_session_object-status
   */
  status?: string;

  /**
   * URL to redirect customers to the checkout page (for hosted mode).
   */
  url?: string;

  /**
   * Mode of the checkout session.
   *
   * Currently only "setup" mode is supported for collecting payment methods.
   */
  mode: StripeCheckoutSessionMode;

  /**
   * The cancel URL where customers are redirected if they cancel.
   */
  cancel_url?: string;

  /**
   * The success URL where customers are redirected after completion.
   */
  success_url?: string;

  /**
   * The return URL for embedded sessions after authentication.
   */
  return_url?: string;
}

/**
 * Stripe Checkout Session mode.
 *
 * Determines the primary purpose of the checkout session.
 */
@friendlyName("BillingAppStripeCheckoutSessionMode")
enum StripeCheckoutSessionMode {
  /**
   * Collect payment method information for later use.
   *
   * Used for subscription billing where the payment method is charged later.
   */
  Setup: "setup",
}
