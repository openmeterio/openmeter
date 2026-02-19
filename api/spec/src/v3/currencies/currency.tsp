import "@typespec/openapi";
import "@typespec/openapi3";
import "../shared/index.tsp";

import "./base.tsp";
import "./custom.tsp";

using TypeSpec.OpenAPI;

namespace Currencies;

/**
 * Currency describes a currency supported by the billing system.
 */
@friendlyName("BillingCurrency")
model Currency {
  ...CurrencyBase;

  /**
   * The currency code. For standard currencies, it should be a standard ISO 4217 code representing the currency.
   * For custom currencies, it should be a unique code based on the ISO 4217 format but not conflicting with any existing standard currency codes.
   */
  @extension("x-go-type", "string")
  @visibility(Lifecycle.Read)
  code: Shared.CurrencyCode | CustomCurrencyCode;

  /**
   * Indicates whether the currency is a custom currency created by the user or a standard currency provided by the system.
   * This field is read-only and is determined by the system based on the currency code.
   * If the currency code matches a standard ISO 4217 code, is_custom will be false. If the currency code is a unique code that does not match any standard ISO 4217 code, is_custom will be true.
   */
  @visibility(Lifecycle.Read)
  is_custom: boolean;
}
