import "./billing.tsp";

/**
 * A customer is a configuration that defines how to match and aggregate events.
 */
@friendlyName("Customer")
@example(#{
  id: "01G65Z755AFWAKHE12NY0CQ9FH",
  key: "my_customer_key",
  usageAttribution: {
      subjects: ["my_subject_key"]
  },
  billingProvider: "stripe",
})
model Customer {
  /**
   * A unique identifier for the customer.
   */
  @visibility("read")
  @example("01G65Z755AFWAKHE12NY0CQ9FH")
  @summary("ID")
  id: ULID;

  /**
   * A unique, human-readable identifier for the customer.
   * Must consist only alphanumeric and underscore characters.
   */
  @example("my_customer_key")
  @summary("Key")
  key: Key;

  // Mapping to attribute metered usage to the customer
  // One customer can have multiple subjects,
  // but one subject can only belong to one customer.
  @summary("Usage Attribution")
  usageAttribution: {
    // TODO allow multiple subjects
    @minItems(1)
    @maxItems(1)
    subjects: Key[];
  };

  /**
   * The primary email address of the customer.
   */
  @summary("Primary Email")
  primaryEmail: string;

  /**
   * Billing configuration for the customer.
   */
  @summary("Billing Configuration")
  billingConfig: {
    taxProvider?: TaxProvider;
    invocingProvider?: InvoicingProvider;
    paymentProvider?: PaymentProvider;
  }

  /**
   * Invoice configuration for the customer.
   */
  @summary("Invoice Configuration")
  invoiceConfig?: {
    address: Address;
    currency: CurrencyCode;
  };

  /**
   * External mappings for the customer.
   */
  @summary("External Mappings")
  external: {
    /**
     * The Stripe customer ID.
     * Mapping to a Stripe Customer object.
     * Required to use Stripe as an invocing provider.
     */
    @summary("Stripe Customer")
    stripeCustomerId?: string;
  };

  /**
   * Additional metadata for the customer.
   */
  @summary("Metadata")
  metadata?: Metadata = #{};
}

/**
 * Address of a customer.
 */
model Address {
  country: CountryCode;
  postalCode?: string;
  state?: string;
  city?: string;
  line1?: string;
  line2?: string;
  phoneNumber?: string;
}
