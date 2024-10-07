import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

import "./app.tsp";
import "./capability.tsp";
import "../errors.tsp";
import "../oauth.tsp";
import "../pagination.tsp";
import "../types.tsp";

using OpenMeter.App;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.App;

/**
 * Installed Apps API.
 */
@route("/api/v1/apps/{id}/stripe")
@tag("AppsStripe")
interface AppStripeAPI {
  /**
   * Stripe webhook.
   */
  @post
  @route("/webhook")
  @operationId("appStripeWebhook")
  appStripeWebhook(@path id: string, @body body: StripeWebhookEvent): StripeWebhookResponse | CommonErrors;
}

/**
 * Stripe webhook response.
 */
@friendlyName("StripeWebhookResponse")
model StripeWebhookResponse {
  namespaceId: string;
  appId: string;
  customerId?: string;
}

/**
 * A installed Stripe app object.
 */
@friendlyName("StripeApp")
@example(#{
  id: "01G65Z755AFWAKHE12NY0CQ9FH",
  type: AppType.Stripe,
  name: "Stripe",
  status: AppStatus.Ready,
  listing: #{
    key: "stripe",
    type: AppType.Stripe,
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
  type: AppType.Stripe;

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
  data: StripeSetupIntent;
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
