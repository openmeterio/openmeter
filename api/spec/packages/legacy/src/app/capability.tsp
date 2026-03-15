import "../types.tsp";

namespace OpenMeter;

/**
 * App capability.
 *
 * Capabilities only exist in config so they don't extend the Resource model.
 */
@friendlyName("AppCapability")
@example(#{
  type: AppCapabilityType.CollectPayments,
  key: "stripe_collect_payment",
  name: "Collect Payments",
  description: "Stripe payments collects outstanding revenue with Stripe customer's default payment method.",
})
model AppCapability {
  /**
   * The capability type.
   */
  type: AppCapabilityType;

  /**
   * Key
   */
  key: Key;

  /**
   * The capability name.
   */
  name: string;

  /**
   * The capability description.
   */
  description: string;
}

/**
 * App capability type.
 */
@friendlyName("AppCapabilityType")
enum AppCapabilityType {
  /**
   * The app can report aggregated usage.
   */
  ReportUsage: "reportUsage",

  /**
   * The app can report raw events.
   */
  ReportEvents: "reportEvents",

  /**
   * The app can calculate tax.
   */
  CalculateTax: "calculateTax",

  /**
   * The app can invoice customers.
   */
  InvoiceCustomers: "invoiceCustomers",

  /**
   * The app can collect payments.
   */
  CollectPayments: "collectPayments",
}
