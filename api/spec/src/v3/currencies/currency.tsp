import "@typespec/openapi";
import "@typespec/openapi3";
import "../shared/index.tsp";

import "./base.tsp";

using TypeSpec.OpenAPI;

namespace Currencies;

/**
 * Currency describes a currency supported by the billing system.
 */
@friendlyName("BillingCurrency")
model Currency {
  ...CurrencyBase;

  /**
   * Standard ISO 4217 code.
   */
  @extension("x-go-type", "string")
  @visibility(Lifecycle.Read)
  code: Shared.CurrencyCode | CustomCurrencyCode;

  /**
   * Flag for the currency if it is custom or not
   */
  @visibility(Lifecycle.Read)
  is_custom: boolean;
}
