import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace Invoice.Pay;

@summary("DueDate contains an amount that should be paid by the given date.")
@friendlyName("BillingDueDate")
model DueDate {
  ...OpenMeter.DueDate;

  @summary("Other details to take into account for the due date.")
  notes?: string;

  @summary("How much needs to be paid by the date.")
  amount: Numeric;

  @summary("Percentage of the total that should be paid by the date.")
  percent?: Percentage;

  @summary("If different from the parent document's base currency.")
  currency?: Currency.Code;
}

namespace OpenMeter {
  @friendlyName("BillingDueDateOpenMeter")
  model DueDate {
    @summary("When the payment is due.")
    // Note: This is a date-time, but we should use a date-time string to be more specific. (original schema has a Date field)
    when: DateTime;
  }
}

@summary("PaymentTermKey defines the type of terms to be applied.")
@friendlyName("BillingPaymentTermKey")
enum PaymentTermKey {
  @summary("Not yet defined")
  na: "",

  @summary("End of month")
  end_of_month: "end-of-month",

  @summary("Due on a specific date")
  due_date: "due-date",

  @summary("Deferred until after the due date")
  deferred: "deferred",

  @summary("Month after the present")
  proximo: "proximo",

  @summary("On receipt of invoice")
  instant: "instant",

  @summary("Chosen by the buyer")
  elective: "elective",

  @summary("Seller to advise buyer in separate transaction")
  pending: "pending",

  @summary("Payment made in advance")
  advanced: "advanced",

  @summary("Payment on Delivery")
  delivery: "delivery",
}

@summary("Terms defines when we expect the customer to pay, or have paid, for the contents of the document.")
@friendlyName("BillingPaymentTerms")
model Terms {
  @summary("Type of terms to be applied.")
  key?: PaymentTermKey;

  @summary("Text detail of the chosen payment terms.")
  detail?: string;

  @summary("Set of dates for agreed payments.")
  due_dates?: DueDate[];

  @summary("Description of the conditions for payment.")
  notes?: string;
}
