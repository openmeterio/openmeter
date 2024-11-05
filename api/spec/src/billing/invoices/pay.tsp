import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace Invoices;

@summary("DueDate contains an amount that should be paid by the given date.")
@friendlyName("BillingDueDate")
model DueDate {
  @summary("When the payment is due.")
  // Note: This is a date-time, but we should use a date-time string to be more specific. (original schema has a Date field)
  dueAt: DateTime;

  @summary("Other details to take into account for the due date.")
  notes?: string;

  @summary("How much needs to be paid by the date.")
  amount: Money;

  @summary("Percentage of the total that should be paid by the date.")
  percent?: Percentage;

  @summary("If different from the parent document's base currency.")
  currency?: CurrencyCode;
}

/*
 * Most of the allowed values are omited, please see https://docs.gobl.org/draft-0/pay/terms
 */
@summary("PaymentPaymentTermType defines the type of terms to be applied.")
@friendlyName("BillingPaymentTermType")
enum PaymentTermType {
  @summary("Due on a specific date")
  dueDate: "due_date",

  @summary("On receipt of invoice")
  instant: "instant",
}

@summary("Terms defines when we expect the customer to pay, or have paid, for the contents of the document.")
@friendlyName("BillingGenericPaymentTerms")
@discriminator("type")
model GenericPaymentTerms<T extends PaymentTermType> {
  @summary("Type of terms to be applied.")
  type: T;

  @summary("Text detail of the chosen payment terms.")
  detail?: string;

  @summary("Description of the conditions for payment.")
  notes?: string;
}

@summary("PaymentTermInstant defines the terms for payment on receipt of invoice.")
@friendlyName("BillingPaymentTermInstant")
model PaymentTermInstant {
  ...GenericPaymentTerms<PaymentTermType.instant>;
}

@summary("PaymentTermDueDate defines the terms for payment on a specific date.")
@friendlyName("BillingPaymentTermDueDate")
model PaymentTermDueDate {
  ...GenericPaymentTerms<PaymentTermType.dueDate>;

  @summary("When the payment is due.")
  @minItems(1)
  dueAt: DueDate[];
}

@summary("PaymentTerms defines the terms for payment.")
@friendlyName("BillingPaymentTerms")
union PaymentTerms {
  instant: PaymentTermInstant,
  dueDate: PaymentTermDueDate,
}
