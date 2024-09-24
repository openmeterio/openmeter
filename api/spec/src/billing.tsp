namespace OpenMeter;

/**
 * A tax config.
 */
@friendlyName("TaxConfig")
model TaxConfig {
  /**
   * Stripe tax config.
   */
  @summary("Stripe tax config")
  stripe?: {
    @summary("Tax code")
    @pattern("^txcd_\\d{8}$")
    @example("txcd_10000000")
    code: string;
  };
}

/**
 * A tax provider.
 */
@friendlyName("TaxProvider")
union TaxProvider {
  /**
   * OpenMeter test billing provider.
   * Useful to generate test invoices.
   */
  @summary("OpenMeter Test")
  openmeter_sandbox: "openmeter_sandbox",

  /**
   * Stripe Tax.
   * See: https://stripe.com/tax
   */
  @summary("Stripe Tax")
  stripe_invoice: "stripe_tax",
}

/**
 * A invoicing provider.
 */
@friendlyName("InvoicingProvider")
union InvoicingProvider {
  /**
   * OpenMeter test billing provider.
   * Useful to generate test invoices.
   */
  @summary("OpenMeter Test")
  openmeter_sandbox: "openmeter_sandbox",

  /**
   * Stripe Invoicing.
   * See: https://stripe.com/invoicing
   */
  @summary("Stripe Invoicing")
  stripe_invoice: "stripe_invoicing",
}

/**
 * A payment provider.
 */
@friendlyName("PaymentProvider")
union PaymentProvider {
  /**
   * OpenMeter test billing provider.
   * Useful to generate test invoices.
   */
  @summary("OpenMeter Test")
  openmeter_sandbox: "openmeter_sandbox",

  /**
   * Stripe Payments.
   * Must be used with Stripe as the invoicing provider.
   * See: https://stripe.com/payments
   */
  @summary("Stripe Payments")
  stripe_payments: "stripe_payments",
}
