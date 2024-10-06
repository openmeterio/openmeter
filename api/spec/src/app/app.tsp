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
@route("/api/v1/apps")
@tag("Apps")
interface AppAPI {
  /**
   * List apps.
   */
  @get
  @route("/")
  @operationId("listApps")
  listApps(...ListAppsRequest): Paginated<App>[] | CommonErrors;

  /**
   * Get the app.
   */
  @get
  @route("/{id}")
  @operationId("getApp")
  getApp(@path id: string): App | CommonErrors;

  /**
   * Uninstall an app.
   */
  @delete
  @route("/{id}")
  @operationId("uninstallApp")
  uninstallApp(@path id: string): App | NotFoundError | CommonErrors;

  /**
   * Stripe webhook.
   */
  @post
  @route("/{id}/stripe/webhook")
  @operationId("handleStripeWebhook")
  handleStripeWebhook(@path id: string, @body body: StripeWebhookEvent): void | CommonErrors;
}

/**
 * Query params for listing installed apps
 */
@friendlyName("ListAppsRequest")
model ListAppsRequest extends PaginatedQuery {}

/**
 * App type.
 */
enum AppType {
  Stripe: "stripe",
}

/**
 * App.
 * One of: stripe
 */
@friendlyName("App")
@discriminator("type")
union App {
  @summary("Stripe")
  stripe: StripeApp,
}

/**
 * Abstract base model for installed apps.
 *
 * Represent an app installed to the organization.
 * This is an actual instance, with its own configuration and credentials.
 */
@friendlyName("AppBase")
model AppBase {
  ...Resource;

  /**
   * The marketplace listing that this installed app is based on.
   */
  @visibility("read")
  listing: Listing;

  /**
   * Status of the app connection.
   */
  @visibility("read")
  status: AppStatus;
}

/**
 * App installed status.
 */
enum AppStatus {
  /**
   * The app is ready to be used.
   */
  Ready: "ready",

  /**
   * The app is unauthorized.
   * This usually happens when the app's credentials are revoked or expired.
   * To resolve this, the user must re-authorize the app.
   */
  Unauthorized: "unauthorized",
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
