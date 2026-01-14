import "../shared/index.tsp";

namespace Apps;

/**
 * Supported capability types for an App.
 *
 * Each capability defines an integration function that an App can perform.
 */
@friendlyName("BillingAppCapabilityType")
enum AppCapabilityType {
  /**
   * The app can report aggregated usage.
   */
  ReportUsage: "report_usage",

  /**
   * The app can report raw events.
   */
  ReportEvents: "report_events",

  /**
   * The app can calculate tax for invoices.
   */
  CalculateTax: "calculate_tax",

  /**
   * The app can issue invoices to customers.
   */
  InvoiceCustomers: "invoice_customers",

  /**
   * The app can collect payments from customers.
   */
  CollectPayments: "collect_payments",
}

/**
 * App capability describes a function that an App can perform.
 */
@friendlyName("BillingAppCapability")
@example(#{
  type: AppCapabilityType.CollectPayments,
  key: "stripe_collect_payments",
  name: "Collect Payments",
  description: "Stripe Payments collects outstanding revenue with Stripe customer's default payment method.",
})
model AppCapability {
  /**
   * Type of the capability.
   */
  type: AppCapabilityType;

  /**
   * Key of the capability.
   */
  key: Shared.ResourceKey;

  /**
   * Name of the capability.
   */
  name: string;

  /**
   * Description of the capability.
   */
  description: string;
}
