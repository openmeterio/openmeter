import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

// OpenMeterCBC is the namespace for the OpenMeter CBC extensions.
namespace Invoice.OpenMeterCBC;

@friendlyName("BillingNoteKey")
enum NoteKey {
  goods: "goods",
  payment: "payment",
  legal: "legal",
  dangerous_goods: "dangerous-goods",
  ack: "ack",
  rate: "rate",
  reason: "reason",
  dispute: "dispute",
  customer: "customer",
  glossary: "glossary",
  customs: "customs",
  general: "general",
  handling: "handling",
  packaging: "packaging",
  loading: "loading",
  price: "price",
  priority: "priority",
  regulatory: "regulatory",
  safety: "safety",
  ship_line: "ship-line",
  supplier: "supplier",
  transport: "transport",
  delivery: "delivery",
  quarantine: "quarantine",
  tax: "tax",

  // openmeter extensions
  invoice_workflow: "invoice-workflow",
}

@discriminator("key")
@friendlyName("BillingBaseNote")
model BaseNote {
  @summary("Key specifying subject of the text")
  key?: NoteKey;

  @summary("Code used for additional data that may be required to identify the note.")
  code?: string;

  @summary("Source of this note, especially useful when auto-generated.")
  src?: Key;

  @summary("The contents of the note")
  text: string;

  @summary("Additional information about the note")
  meta?: CBC.Meta;
}

/**
 * ReasonNote is a note that contains information about the reason while this item exists.
 *
 * Can be used for unstructured information about the reason.
 */
@friendlyName("BillingReasonNote")
model ReasonNote extends BaseNote {
  key: NoteKey.reason;
}

/**
 * InvoiceWorkflowNote is a note that contains information about the invoice workflow.
 */
@friendlyName("BillingInvoiceWorkflowNote")
// TODO: we should rather represent this as static members!
model InvoiceWorkflowNote extends BaseNote {
  key: NoteKey.invoice_workflow;

  @summary("The details of the invoice workflow")
  workflow: ExistingInvoiceWorkflow;
}

alias InvoiceNotes = ReasonNote | InvoiceWorkflowNote;

/**
 * LineNote is a note that contains information about a line item.
 */
alias LineNotes = ReasonNote;

/**
 * LineItemNote is a note that contains information about an item of a line.
 */
alias LineItemNotes = ReasonNote;

@friendlyName("BillingExistingInvoiceWorkflow")
model ExistingInvoiceWorkflow {
  /**
   * updatedAt is the time when the invoice workflow was last updated.
   *
   * This field is updated when the invoice workflow is updated, during put requests we
   * are validating the update time to ensure there is no parallel updates happening.
   */
  updatedAt: DateTime;

  @summary("Should the invoice auto advance to the next status?")
  autoAdvance: boolean;

  /**
   * The invoice stays in the draft status until this time.
   *
   * If auto advance is disabled the value will be empty.
   */
  draftUtil?: DateTime;

  /**
   * The period after which the invoice is due.
   *
   * default: 7 days
   */
  @encode(DurationKnownEncoding.ISO8601)
  @example(duration.fromISO("P1D"))
  dueAfter?: duration;
}
