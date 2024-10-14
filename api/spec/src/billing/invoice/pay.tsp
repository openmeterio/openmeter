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

/*
 * Most of the allowed values are omited, please see https://docs.gobl.org/draft-0/pay/terms
 */
@summary("PaymentTermKey defines the type of terms to be applied.")
@friendlyName("BillingPaymentTermKey")
enum PaymentTermKey {
  @summary("Not yet defined")
  na: "",

  @summary("End of month")
  endOfMonth: "end_of_month",

  @summary("Due on a specific date")
  dueDate: "due_date",

  @summary("On receipt of invoice")
  instant: "instant",
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
