import "./properties.tsp";

namespace Shared;

/**
 * Address
 */
@friendlyName("Address")
model Address {
  /**
   * Country code in [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 format.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read, Lifecycle.Update)
  @summary("Country")
  country?: CountryCode;

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
