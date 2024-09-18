import "./billing.tsp";
import "./types.tsp";

namespace OpenMeter;

/**
 * A customer object.
 */
@friendlyName("Customer")
model Customer extends Resource {
  /**
   * A unique identifier for the customer.
   */
  @visibility("read")
  @example("01G65Z755AFWAKHE12NY0CQ9FH")
  @summary("ID")
  id: ULID;

  // Mapping to attribute metered usage to the customer
  @summary("Usage Attribution")
  usageAttribution: CustomerUsageAttribution;

  /**
   * The primary email address of the customer.
   */
  @summary("Primary Email")
  primaryEmail?: string;

  /**
   * Currency of the customer.
   * Used for billing, tax and invoicing.
   */
  @summary("Currency")
  currency?: CurrencyCode;

  /**
   * The billing address of the customer.
   * Used for tax and invoicing.
   */
  @summary("Billing Address")
  billingAddress?: Address;

  /**
   * The tax provider for the customer.
   */
  @summary("Tax Provider")
  taxProvider?: TaxProvider;

  /**
   * The invoicing provider for the customer.
   */
  @summary("Invoicing Provider")
  invoicingProvider?: InvoicingProvider;

  /**
   * The payment provider for the customer.
   */
  @summary("Payment Provider")
  paymentProvider?: PaymentProvider;

  /**
   * External mappings for the customer.
   */
  @summary("External Mappings")
  external?: CustomerExternalMapping;
}

/**
 * Mapping to attribute metered usage to the customer.
 * One customer can have multiple subjects,
 * but one subject can only belong to one customer.
 */
@friendlyName("Usage Attribution")
model CustomerUsageAttribution {
  /**
   * The subjects that are attributed to the customer.
   * @TODO allow multiple subjects
   */
  @summary("Subjects")
  @minItems(1)
  @maxItems(1)
  subjects: Key[];
}

/**
 * External mappings for the customer.
 */
@friendlyName("External Mapping")
model CustomerExternalMapping {
  /**
   * The Stripe customer ID.
   * Mapping to a Stripe Customer object.
   * Required to use Stripe as an invocing provider.
   */
  @summary("Stripe Customer")
  stripeCustomerId?: string;
}
