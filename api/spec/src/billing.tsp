import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter;

/**
 * A tax config.
 */
@friendlyName("TaxConfig")
model TaxConfig {
  /**
   * Stripe tax config.
   */
  @summary("Stripe tax config")
  stripe?: {
    @summary("Tax code")
    @pattern("^txcd_\\d{8}$")
    @example("txcd_10000000")
    code: string;
  };
}

/**
 * A tax provider.
 */
@friendlyName("TaxProvider")
union TaxProvider {
  /**
   * OpenMeter test billing provider.
   * Useful to generate test invoices.
   */
  @summary("OpenMeter Test")
  openmeter_sandbox: "openmeter_sandbox",

  /**
   * Stripe Tax.
   * See: https://stripe.com/tax
   */
  @summary("Stripe Tax")
  stripe_invoice: "stripe_tax",
}

/**
 * A invoicing provider.
 */
@friendlyName("InvoicingProvider")
union InvoicingProvider {
  /**
   * OpenMeter test billing provider.
   * Useful to generate test invoices.
   */
  @summary("OpenMeter Test")
  openmeter_sandbox: "openmeter_sandbox",

  /**
   * Stripe Invoicing.
   * See: https://stripe.com/invoicing
   */
  @summary("Stripe Invoicing")
  stripe_invoice: "stripe_invoicing",
}

/**
 * A payment provider.
 */
@friendlyName("PaymentProvider")
union PaymentProvider {
  /**
   * OpenMeter test billing provider.
   * Useful to generate test invoices.
   */
  @summary("OpenMeter Test")
  openmeter_sandbox: "openmeter_sandbox",

  /**
   * Stripe Payments.
   * Must be used with Stripe as the invoicing provider.
   * See: https://stripe.com/payments
   */
  @summary("Stripe Payments")
  stripe_payments: "stripe_payments",
}

namespace Billing {


@route("/api/v1/billing/invoices")
interface InvoiceEndpoints {
  /**
   * List invoices
   */
  @get
  @summary("List invoices")
  @operationId("billingListInvoices")
  list(
    @query
    @summary("Filter by the customer ID or key")
    customers?: Array<ULIDOrKey>,

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryLimitOffset,
    ...OpenMeter.QueryOrdering<InvoiceOrderBy>,
  ): OpenMeter.PaginatedResponse<ULIDOrKey> | OpenMeter.CommonErrors;
}

// Just to ensure that this is not due to different values, it explicitly matches the
// NotificationRuleOrderBy enum, but the same issue happens with other names.
@friendlyName("BillingInvoiceOrderBy")
enum InvoiceOrderBy { // at notification this matches the friendly name but it doesn't matter
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  id: "id",
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  type: "type",
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  createdAt: "createdAt",
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  updatedAt: "updatedAt",
}

}
