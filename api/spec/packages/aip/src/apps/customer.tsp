import "./app.tsp";

namespace Apps;

/**
 * App customer data.
 */
@friendlyName("BillingAppCustomerData")
model AppCustomerData {
  /**
   * Used if the customer has a linked Stripe app.
   *
   * Set to `null` to remove the Stripe data of the customer; omit to leave it
   * unchanged.
   */
  @summary("Stripe")
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  stripe?: Apps.AppCustomerDataStripe | null;

  /**
   * Used if the customer has a linked external invoicing app.
   *
   * Set to `null` to remove the external invoicing data of the customer; omit to
   * leave it unchanged.
   */
  @summary("External invoicing")
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  external_invoicing?: Apps.AppCustomerDataExternalInvoicing | null;
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

  /**
   * Labels for this Stripe integration on the customer.
   */
  @summary("Labels")
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  labels?: Common.Labels;
}

/**
 * External invoicing customer data.
 */
@friendlyName("BillingAppCustomerDataExternalInvoicing")
model AppCustomerDataExternalInvoicing {
  /**
   * Labels for this external invoicing integration on the customer.
   */
  @summary("Labels")
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  labels?: Common.Labels;
}
