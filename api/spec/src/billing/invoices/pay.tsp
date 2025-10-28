import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter;

/**
 * PaymentDueDate contains an amount that should be paid by the given date.
 */
@friendlyName("PaymentDueDate")
model PaymentDueDate {
  /**
   * When the payment is due.
   */
  // Note: This is a date-time, but we should use a date-time string to be more specific. (original schema has a Date field)
  @visibility(Lifecycle.Read)
  dueAt: DateTime;

  /**
   * Other details to take into account for the due date.
   */
  @visibility(Lifecycle.Read)
  notes?: string;

  /**
   * How much needs to be paid by the date.
   */
  @visibility(Lifecycle.Read)
  amount: Money;

  /**
   * Percentage of the total that should be paid by the date.
   */
  @visibility(Lifecycle.Read)
  percent?: Percentage;

  /**
   * If different from the parent document's base currency.
   */
  @visibility(Lifecycle.Read)
  currency?: CurrencyCode;
}

/*
 * Most of the allowed values are omited, please see https://docs.gobl.org/draft-0/pay/terms
 */
/**
 * PaymentTermType defines the type of terms to be applied.
 */
@friendlyName("PaymentTermType")
enum PaymentTermType {
  /**
   * Due on a specific date.
   */
  dueDate: "due_date",

  /**
   * On receipt of invoice
   */
  instant: "instant",
}

@summary("Terms defines when we expect the customer to pay, or have paid, for the contents of the document.")
@friendlyName("GenericPaymentTerms")
model GenericPaymentTerms<T extends PaymentTermType> {
  /**
   * Type of terms to be applied.
   */
  type: T;

  /**
   * Text detail of the chosen payment terms.
   */
  @visibility(Lifecycle.Read)
  detail?: string;

  /**
   * Description of the conditions for payment.
   */
  @visibility(Lifecycle.Read)
  notes?: string;
}

/**
 * PaymentTermInstant defines the terms for payment on receipt of invoice.
 */
@friendlyName("PaymentTermInstant")
model PaymentTermInstant {
  ...GenericPaymentTerms<PaymentTermType.instant>;
}

/**
 * PaymentTermDueDate defines the terms for payment on a specific date.
 */
@friendlyName("PaymentTermDueDate")
model PaymentTermDueDate {
  ...GenericPaymentTerms<PaymentTermType.dueDate>;

  /**
   * When the payment is due.
   */
  @minItems(1)
  @visibility(Lifecycle.Read)
  dueAt: PaymentDueDate[];
}

/**
 * PaymentTerms defines the terms for payment.
 */
@friendlyName("PaymentTerms")
union PaymentTerms {
  instant: PaymentTermInstant,
  dueDate: PaymentTermDueDate,
}
