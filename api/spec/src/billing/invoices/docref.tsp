import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace Invoices;

/**
 * DocumentRefType defines the type of document that is being referenced.
 */
@friendlyName("BillingDocumentRefType")
enum DocumentRefType {
  creditNoteOriginalInvoice: "credit_node_original_invoice",
}

/**
 * Omitted fields:
 *  period: Tax period in which the referred document had an effect required by some tax regimes and formats.
 *  stamps: Seals of approval from other organisations that may need to be listed.
 *  ext: 	Extensions for additional codes that may be required.
 */
@summary("DocumentRef is used to describe an existing document or a specific part of it's contents.")
@friendlyName("BillingGenericDocumentRef")
model GenericDocumentRef {
  @summary("Type of the document referenced.")
  @visibility("read", "query")
  type: DocumentRefType;

  @summary("Human readable description on why this reference is here or needs to be used.")
  reason?: string;

  @summary("Additional details about the document.")
  description?: string;
}

/**
 * CreditNoteOriginalInvoiceRef is used to reference the original invoice that a credit note is based on.
 */
@friendlyName("BillingCreditNoteOriginalInvoiceRef")
model CreditNoteOriginalInvoiceRef extends GenericDocumentRef {
  type: DocumentRefType.creditNoteOriginalInvoice;

  @summary("IssueAt reflects the time the document was issued.")
  issuedAt?: DateTime;

  @summary("(Serial) Number of the referenced document.")
  @visibility("read", "query")
  number?: InvoiceNumber;

  @summary("Link to the source document.")
  @format("uri")
  url: string;
}

/**
 * DocumentRef is used to describe a reference to an existing document (invoice).
 */
@friendlyName("BillingDocumentRef")
union DocumentRef {
  creditNoteOriginalInvoice: CreditNoteOriginalInvoiceRef,
}
