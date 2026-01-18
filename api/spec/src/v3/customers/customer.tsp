import "../shared/index.tsp";
import "../common/properties.tsp";

namespace Customers;

/**
 * Customers can be individuals or organizations that can subscribe to plans and have access to features.
 */
@friendlyName("BillingCustomer")
model Customer {
  ...Shared.Resource;

  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  @visibility(Lifecycle.Create, Lifecycle.Read)
  key: Shared.ExternalResourceKey;

  /**
   * Mapping to attribute metered usage to the customer by the event subject.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read, Lifecycle.Update)
  @summary("Usage Attribution")
  usage_attribution?: CustomerUsageAttribution;

  /**
   * The primary email address of the customer.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read, Lifecycle.Update)
  @summary("Primary Email")
  primary_email?: string;

  /**
   * Currency of the customer.
   * Used for billing, tax and invoicing.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read, Lifecycle.Update)
  @summary("Currency")
  currency?: Shared.CurrencyCode;

  /**
   * The billing address of the customer.
   * Used for tax and invoicing.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read, Lifecycle.Update)
  @summary("Billing Address")
  billing_address?: Address;
}

/**
 * Address
 */
@friendlyName("BillingAddress")
model Address {
  /**
   * Country code in [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 format.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read, Lifecycle.Update)
  @summary("Country")
  country?: Shared.CountryCode;

  /**
   * Postal code.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read, Lifecycle.Update)
  @summary("Postal Code")
  postal_code?: string;

  /**
   * State or province.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read, Lifecycle.Update)
  @summary("State")
  state?: string;

  /**
   * City.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read, Lifecycle.Update)
  @summary("City")
  city?: string;

  /**
   * First line of the address.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read, Lifecycle.Update)
  @summary("Line 1")
  line1?: string;

  /**
   * Second line of the address.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read, Lifecycle.Update)
  @summary("Line 2")
  line2?: string;

  /**
   * Phone number.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read, Lifecycle.Update)
  @summary("Phone Number")
  phone_number?: string;
}

/**
 * Mapping to attribute metered usage to the customer.
 * One customer can have zero or more subjects,
 * but one subject can only belong to one customer.
 */
@friendlyName("BillingCustomerUsageAttribution")
model CustomerUsageAttribution {
  /**
   * The subjects that are attributed to the customer.
   * Can be empty when no usage event subjects are associated with the customer.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read, Lifecycle.Update)
  @summary("Subject Keys")
  @minItems(0)
  subject_keys: UsageAttributionSubjectKey[];
}

/**
 * Subject key.
 */
@minLength(1)
@friendlyName("UsageAttributionSubjectKey")
scalar UsageAttributionSubjectKey extends string;

/**
 * Customer reference.
 */
@friendlyName("BillingCustomerReference")
model CustomerReference {
  /**
   * The ID of the customer.
   */
  id: Shared.ULID;
}

/**
 * Customer reference by external key.
 */
@friendlyName("BillingCustomerKeyReference")
model CustomerKeyReference {
  /**
   * The external key of the customer.
   */
  key: Shared.ExternalResourceKey;
}
