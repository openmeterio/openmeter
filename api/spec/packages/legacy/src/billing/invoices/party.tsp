import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

import "../../customer";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter;

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
/**
 * Party represents a person or business entity.
 */
@friendlyName("BillingParty")
model BillingParty {
  /**
   * Unique identifier for the party (if available)
   */
  @visibility(Lifecycle.Read)
  id?: string;

  /**
   * An optional unique key of the party (if available)
   */
  @summary("Key")
  key?: ExternalKey;

  /**
   * Legal name or representation of the organization.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  name?: string;

  /**
   * The entity's legal ID code used for tax purposes. They may have
   * other numbers, but we're only interested in those valid for tax purposes.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  taxId?: BillingPartyTaxIdentity;

  /**
   * Regular post addresses for where information should be sent if needed.
   */
  @maxItems(1)
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  addresses?: Address[];
}

/**
 * BillingInvoiceCustomerExtendedDetails is a collection of fields that are used to extend the billing party details for invoices.
 *
 * These fields contain the OpenMeter specific details for the customer, that are not strictly required for the invoice itself.
 */
@friendlyName("BillingInvoiceCustomerExtendedDetails")
model BillingInvoiceCustomerExtendedDetails {
  ...BillingParty;

  /**
   * Mapping to attribute metered usage to the customer
   */
  @summary("Usage Attribution")
  usageAttribution: CustomerUsageAttribution;
}

/**
 * Identity stores the details required to identify an entity for tax purposes in a specific country.
 */
@friendlyName("BillingPartyTaxIdentity")
model BillingPartyTaxIdentity {
  /**
   * Normalized tax code shown on the original identity document.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  code?: TaxIdentificationCode;
}
