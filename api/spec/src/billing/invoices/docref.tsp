import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter;

/**
 * InvoiceDocumentRefType defines the type of document that is being referenced.
 */
@friendlyName("InvoiceDocumentRefType")
enum InvoiceDocumentRefType {
  creditNoteOriginalInvoice: "credit_note_original_invoice",
}

/**
 * Omitted fields:
 *  period: Tax period in which the referred document had an effect required by some tax regimes and formats.
 *  stamps: Seals of approval from other organisations that may need to be listed.
 *  ext: 	Extensions for additional codes that may be required.
 */
@summary("InvoiceGenericDocumentRef is used to describe an existing document or a specific part of it's contents.")
@friendlyName("InvoiceGenericDocumentRef")
model InvoiceGenericDocumentRef {
  /**
   * Type of the document referenced.
   */
  @visibility(Lifecycle.Read)
  type: InvoiceDocumentRefType;

  /**
   * Human readable description on why this reference is here or needs to be used.
   */
  @visibility(Lifecycle.Read)
  reason?: string;

  /**
   * Additional details about the document.
   */
  @visibility(Lifecycle.Read)
  description?: string;
}

/**
 * CreditNoteOriginalInvoiceRef is used to reference the original invoice that a credit note is based on.
 */
@friendlyName("CreditNoteOriginalInvoiceRef")
model InvoiceCreditNoteOriginalInvoiceRef extends InvoiceGenericDocumentRef {
  /**
   * Type of the invoice.
   */
  type: InvoiceDocumentRefType.creditNoteOriginalInvoice;

  /**
   * IssueAt reflects the time the document was issued.
   */
  @visibility(Lifecycle.Read)
  issuedAt?: DateTime;

  /**
   * (Serial) Number of the referenced document.
   */
  @visibility(Lifecycle.Read)
  number?: InvoiceNumber;

  /**
   * Link to the source document.
   */
  @format("uri")
  @visibility(Lifecycle.Read)
  url: string;
}

/**
 * InvoiceDocumentRef is used to describe a reference to an existing document (invoice).
 */
@friendlyName("InvoiceDocumentRef")
union InvoiceDocumentRef {
  creditNoteOriginalInvoice: InvoiceCreditNoteOriginalInvoiceRef,
}
