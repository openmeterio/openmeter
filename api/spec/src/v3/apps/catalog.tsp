import "../shared/index.tsp";
import "./capability.tsp";
import "./app.tsp";

namespace Apps;

/**
 * Supported installation methods for an app.
 */
@friendlyName("BillingAppInstallMethods")
enum AppInstallMethods {
  /**
   * Install by completing an OAuth 2.0 authorization flow.
   */
  WithOAuth2: "with_oauth2",

  /**
   * Install by providing an API key.
   */
  WithAPIKey: "with_api_key",

  /**
   * Install without providing any credentials.
   */
  NoCredentialsRequired: "no_credentials_required",
}

/**
 * Available apps for billing integrations to connect with third-party services.
 * Apps can have various capabilities like syncing data from or to external systems, integrating with third-party services for tax calculation, delivery of invoices, collection of payments, etc.
 */
@friendlyName("BillingAppCatalogItem")
@example(#{
  type: AppType.Stripe,
  name: "Stripe",
  description: "Stripe interation allows you to collect payments with Stripe.",
  capabilities: #[
    #{
      type: AppCapabilityType.CalculateTax,
      key: "stripe_calculate_tax",
      name: "Calculate Tax",
      description: "Stripe Tax calculates tax portion of the invoices.",
    },
    #{
      type: AppCapabilityType.InvoiceCustomers,
      key: "stripe_invoice_customers",
      name: "Invoice Customers",
      description: "Stripe invoices customers with due amount.",
    },
    #{
      type: AppCapabilityType.CollectPayments,
      key: "stripe_collect_payments",
      name: "Collect Payments",
      description: "Stripe payments collects outstanding revenue with Stripe customer's default payment method.",
    }
  ],
  install_methods: #[
    AppInstallMethods.WithOAuth2,
    AppInstallMethods.WithAPIKey
  ],
})
model AppCatalogItem {
  /**
   * Type of the app.
   */
  @visibility(Lifecycle.Read)
  type: AppType;

  /**
   * Name of the app.
   */
  @visibility(Lifecycle.Read)
  name: string;

  /**
   * Description of the app.
   */
  @visibility(Lifecycle.Read)
  description: string;

  /**
   * Capabilities of the app.
   */
  @visibility(Lifecycle.Read)
  capabilities: AppCapability[];

  /**
   * Available install methods of the app.
   */
  @visibility(Lifecycle.Read)
  install_methods: AppInstallMethods[];
}
