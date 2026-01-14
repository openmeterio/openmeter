import "./app.tsp";

namespace Apps;

/**
 * App customer data.
 */
@friendlyName("BillingAppCustomerData")
model AppCustomerData {
  /**
   * Used if the customer has a linked Stripe app.
   */
  @summary("Stripe")
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  stripe?: Apps.AppCustomerDataStripe;

  /**
   * Used if the customer has a linked custom invoicing app.
   */
  @summary("Custom invoicing")
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  custom_invoicing?: Apps.AppCustomerDataCustomInvoicing;
}

/**
 * Stripe customer data.
 */
@friendlyName("BillingAppCustomerDataStripe")
model AppCustomerDataStripe {
  /**
   * The Stripe customer ID used.
   */
  @summary("Stripe customer ID")
  @example("cus_1234567890")
  customer_id?: string;

  /**
   * The Stripe default payment method ID.
   */
  @summary("Stripe default payment method ID")
  @example("pm_1234567890")
  default_payment_method_id?: string;
}

/**
 * Custom invoicing customer data.
 */
@friendlyName("BillingAppCustomerDataCustomInvoicing")
model AppCustomerDataCustomInvoicing {
  /**
   * Labels to be used by the custom invoicing provider.
   */
  @summary("Labels")
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  labels?: Common.Labels;
}
