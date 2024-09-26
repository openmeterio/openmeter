import "./app.tsp";
import "./capability.tsp";
import "./connected.tsp";
import "../types.tsp";

using OpenMeter.App;

namespace OpenMeter.App;

/**
 * A connected Stripe app object.
 */
@friendlyName("AppStripe")
@example(#{
  id: "01G65Z755AFWAKHE12NY0CQ9FH",
  type: AppType.Stripe,
  name: "Stripe",
  createdAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
  updatedAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
  status: ConnectedAppStatus.Ready,
  capabilities: #[
    #{
      type: AppCapabilityType.CollectPayments,
      key: "stripe_calculate_tax",
      name: "Calculate Tax",
      description: "Stripe Tax calculates tax portion of the invoices.",
      requirements: #[AppRequirements.CustomerExternalStripeCustomerId],
    },
    #{
      type: AppCapabilityType.CollectPayments,
      key: "stripe_invoice_customers",
      name: "Invoice Customers",
      description: "Stripe invoices customers with due amount.",
      requirements: #[AppRequirements.CustomerExternalStripeCustomerId],
    },
    #{
      type: AppCapabilityType.CollectPayments,
      key: "stripe_collect_payments",
      name: "Collect Payments",
      description: "Stripe payments collects outstanding revenue with Stripe customer's default payment method.",
      requirements: #[AppRequirements.CustomerExternalStripeCustomerId],
    }
  ],

  // Stripe specific fields
  accountId: "acct_123456789",

  livemode: true,
})
model AppConnectedStripe {
  ...ConnectedApp;

  /**
   * The Stripe account ID.
   */
  accountId: string;

  /**
   * Livemode, true if the app is in production mode.
   */
  livemode: boolean;
}
