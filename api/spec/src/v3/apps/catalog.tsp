import "../shared/index.tsp";
import "./app.tsp";

namespace Apps;

/**
 * Available apps for billing integrations to connect with third-party services.
 * Apps can have various capabilities like syncing data from or to external systems, integrating with third-party services for tax calculation, delivery of invoices, collection of payments, etc.
 */
@friendlyName("BillingAppCatalogItem")
@example(#{
  type: AppType.Stripe,
  name: "Stripe",
  description: "Stripe integration allows you to collect payments with Stripe.",
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
}
