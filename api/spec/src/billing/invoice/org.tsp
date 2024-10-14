import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace Invoice.Org;

/*
 * Omitted fields
 *
 *  identities: Set of codes used to identify the party in other systems.
 *    -> This is maintained by apps, and for now we don't see any reason to have this included.
 *  people: Details of physical people who represent the party.
 *    -> We don't need this level of detail for now, the name is enough if we deal with a person.
 *  inboxes: 	Digital inboxes used for forwarding electronic versions of documents
 *    -> We don't support this.
 *  websites: 	Public websites that provide further information about the party.
 *  telephones:   Telephone numbers for contacting the party.
 *  registration: Additional registration details about the company that may need to be included in a document.
 *  logos: Images that can be used to identify the party visually.
 *  ext: (tax) Extension code map for any additional regime specific codes that may be required.
 */
@summary("Party represents a person or business entity.")
@friendlyName("BillingParty")
model Party {
  @summary("Legal name or representation of the organization.")
  name?: string;

  @summary("The entity’s legal ID code used for tax purposes. They may have other numbers, but we’re only interested in those valid for tax purposes.")
  taxId?: Tax.Identity;

  @summary("Regular post addresses for where information should be sent if needed.")
  @maxItems(1)
  addresses?: Array<Org.Address>;

  @summary("Electronic mail addresses")
  @maxItems(1)
  emails?: Array<Org.Email>;

  @summary("Any additional semi-structured information that does not fit into the rest of the party.")
  meta?: CBC.Meta;
}

@summary("Email describes the electronic mailing details.")
@friendlyName("BillingEmail")
model Email {
  @summary("Identifier for the email.")
  label?: string;

  @summary("The email address.")
  addr: string;

  @summary("Additional fields.")
  meta?: CBC.Meta;
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
  country?: L10n.ISOCountryCode;

  @summary("Additional fields.")
  meta?: CBC.Meta;
}

/**
 * Omitted fields:
 *  period: Tax period in which the referred document had an effect required by some tax regimes and formats.
 *  stamps: Seals of approval from other organisations that may need to be listed.
 *  ext: 	Extensions for additional codes that may be required.
 */
@summary("DocumentRef is used to describe an existing document or a specific part of it’s contents.")
@friendlyName("BillingDocumentRef")
model DocumentRef {
  @summary("Type of the document referenced.")
  type?: CBC.Key;

  @summary("IssueAt reflects the time the document was issued.")
  issuedAt?: DateTime;

  @summary("Series the referenced document belongs to.")
  series?: CBC.Code;

  @summary("Source document's code or other identifier.")
  code: CBC.Code;

  @summary("List of additional codes, IDs, or SKUs which can be used to identify the document or its contents, agreed upon by the supplier and customer.")
  identities?: Identity[];

  @summary("Human readable description on why this reference is here or needs to be used.")
  reason?: string;

  @summary("Additional details about the document.")
  description?: string;

  @summary("Link to the source document.")
  @format("uri")
  url?: string;

  @summary("Meta contains additional information about the document.")
  meta?: CBC.Meta;
}

@summary("Identity is used to define a code for a specific context.")
@friendlyName("BillingIdentity")
model Identity {
  @summary("Optional label useful for non-standard identities to give a bit more context.")
  label?: string;

  @summary("Country from which the identity was issued.")
  country?: L10n.ISOCountryCode;

  @summary("Uniquely classify this identity using a key instead of a code.")
  key?: CBC.Key;

  @summary("The type of Code being represented and usually specific for a particular context, country, or tax regime, and cannot be used alongside the key.")
  type?: CBC.Code;

  @summary("The actual value of the identity code.")
  code: CBC.Code;

  @summary("Description adds details about what the code could mean or imply.")
  description?: string;
}

// TODO: we need to have a unit database
@summary("Unit describes how the quantity of the product should be interpreted either using a GOBL key, or UN/ECE code.")
scalar Unit extends string;

/**
 * UnitPriceItem represents a product or service that is sold to the customer based on a unit price.
 */
/* Omitted fields:
 * - alt_prices: 	AltPrices defines a list of prices with their currencies that may be used as an alternative to the item’s base price.
 * - origin: Country code of where this item was from originally.
 * - ext: 	Extension code map for any additional regime specific codes that may be required.
 */
@friendlyName("BillingUnitPriceItem")
model UnitPriceItem {
  @summary("Primary reference code that identifies this item. Additional codes can be provided in the ‘identities’ property.")
  ref?: string;

  @summary("Special key used to classify the item sometimes required by some regimes.")
  key?: CBC.Key;

  @summary("Human readable name of the item.")
  name: string;

  @summary("List of additional codes, IDs, or SKUs which can be used to identify the item. They should be agreed upon between supplier and customer.")
  identities?: Org.Identity[];

  @summary("Description of the item.")
  description?: string;

  @summary("Currency used for the item’s price. Currency is mandatory for all items, so that we can support multi-currency setups.")
  currency: Currency.Code;

  @summary("Base price of a single unit to be sold.")
  price: Numeric;

  @summary("Unit of measure")
  unit?: Org.Unit;

  @summary("Additional metadata information that may be useful.")
  meta?: CBC.Meta;
}
