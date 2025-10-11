import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";

import "../meters.tsp";
import "../errors.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.Billing.Invoice;

/**
 * InvoiceCostHandler is a collection of endpoints that allow getting the cost of an invoice.
 */
@tag("Billing")
interface InvoiceCostHandler {
  /**
   * Get an invoice feature cost.
   */
  @get
  @route("/api/v1/billing/invoices/{invoiceId}/lines/{lineId}/cost")
  @summary("Get an invoice feature cost")
  @operationId("getInvoiceLineCost")
  getInvoiceLineCost(
    @path
    invoiceId: ULID,

    @path
    lineId: string,

    ...GetInvoiceLineCostParam,
  ): InvoiceLineCost | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

/**
 * InvoiceLineCost is the cost of a feature for an invoice.
 */
@friendlyName("InvoiceLineCost")
model InvoiceLineCost {
  /**
   * The start of the period the value is aggregated over.
   */
  from: DateTime;

  /**
   * The end of the period the value is aggregated over.
   */
  to: DateTime;

  /**
   * The cost per unit of the feature for the invoice.
   */
  costPerUnit: Numeric;

  /**
   * The total usage of the feature for the invoice.
   */
  totalUsage: Numeric;

  /**
   * The total cost of the feature for the invoice.
   */
  totalCost: Numeric;

  /**
   * Currency of the cost.
   */
  currency: CurrencyCode;

  /**
   * The rows of the feature cost for the invoice.
   */
  rows: InvoiceLineCostRow[];
}

/**
 * InvoiceLineCostRow is a row of the feature cost for the invoice.
 */
@friendlyName("InvoiceLineCostRow")
model InvoiceLineCostRow {
  /**
   * The start of the window the value is aggregated over.
   */
  windowStart: DateTime;

  /**
   * The end of the window the value is aggregated over.
   */
  windowEnd: DateTime;

  /**
   * The usage of the feature for the invoice.
   */
  usage: Numeric;

  /**
   * The cost of the feature for the invoice.
   */
  cost: Numeric;

  /**
   * The group by values the value is aggregated over.
   */
  groupBy: Record<string | null>;
}

/**
 * GetInvoiceLineCostParam is the parameter for the getInvoiceLineCost endpoint.
 */
@friendlyName("GetInvoiceLineCostParam")
model GetInvoiceLineCostParam {
  /**
   * If not specified, a single usage aggregate will be returned for the entirety of the specified period for each subject and group.
   *
   * For example: ?windowSize=DAY
   */
  @query
  windowSize?: OpenMeter.WindowSize;

  /**
   * The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones).
   * If not specified, the UTC timezone will be used.
   *
   * For example: ?windowTimeZone=UTC
   */
  @query
  @example("America/New_York")
  windowTimeZone?: string = "UTC";

  /**
   * If not specified a single aggregate will be returned for each subject and time window.
   * `subject` is a reserved group by value.
   *
   * For example: ?groupBy=subject&groupBy=model
   */
  @query(#{ explode: true })
  @example(#["model", "type"])
  groupBy?: string[];
}
