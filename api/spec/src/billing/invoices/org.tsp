import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace Invoices;

/*
 * Omitted fields
 *
 *  identities: Set of codes used to identify the party in other systems.
 *    -> This is maintained by apps, and for now we don't see any reason to have this included.
 *  people: Details of physical people who represent the party.
 *    -> We don't need this level of detail for now, the name is enough if we deal with a person.
 *  inboxes: 	Digital inboxes used for forwarding electronic versions of documents
 *    -> We don't support this.
 *  emails:
 *   -> We don't need this on the invoice.
 *  websites: 	Public websites that provide further information about the party.
 *  telephones:   Telephone numbers for contacting the party.
 *  registration: Additional registration details about the company that may need to be included in a document.
 *  logos: Images that can be used to identify the party visually.
 *  ext: (tax) Extension code map for any additional regime specific codes that may be required.
 */
@summary("Party represents a person or business entity.")
@friendlyName("BillingParty")
model Party {
  @summary("ID of the party.")
  @visibility("read", "query", "update")
  id: ULID;

  @summary("Legal name or representation of the organization.")
  name?: string;

  @summary("The entity's legal ID code used for tax purposes. They may have other numbers, but we're only interested in those valid for tax purposes.")
  taxId?: TaxIdentity;

  @summary("Regular post addresses for where information should be sent if needed.")
  @maxItems(1)
  addresses?: Array<Address>;
}

/*
 * Omitted fields:
 * po_box: Post office box number for the address.
 * num: Number of the building in the street.
 * floor: Floor of the building.
 * door: Door number of the building.
 * block: Block number within the building.
 * coordinates: When the postal address is not sufficient, coordinates help locate the address more precisely.
 */
@summary("Address describes the physical location of a party.")
@friendlyName("BillingAddress")
model Address {
  @summary("Identifier for the address.")
  label?: string;

  @summary("The street address.")
  street?: string;

  @summary("Additional street address details.")
  streetExtra?: string;

  @summary("Village, town, district, or city, typically inside a region.")
  locality?: string;

  @summary("Province, county, or state, inside a country.")
  region?: string;

  @summary("Postal code for the address.")
  code?: string;

  @summary("The country.")
  country?: ISOCountryCode;
}
