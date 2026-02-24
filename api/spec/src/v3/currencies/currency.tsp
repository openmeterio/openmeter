import "@typespec/openapi";
import "@typespec/openapi3";
import "../shared/index.tsp";

import "./base.tsp";
import "./custom.tsp";

using TypeSpec.OpenAPI;

namespace Currencies;

/**
 * Currency type for custom currencies.
 * It should be a unique code but not conflicting with any existing standard currency codes.
 */
@friendlyName("BillingCurrencyType")
enum CurrencyType {
  Standard,
  Custom,
}

/**
 * Currency describes a currency supported by the billing system.
 */
@friendlyName("BillingCurrency")
@discriminator("currency_type")
model Currency {
  ...CurrencyBase;

  /**
   * The currency code. For standard currencies, it should be a standard ISO 4217 code representing the currency.
   * For custom currencies, it should be a unique code but not conflicting with any existing standard currency codes.
   */
  @extension("x-go-type", "string")
  @visibility(Lifecycle.Read)
  code: Shared.CurrencyCode | CustomCurrencyCode;

  /**
   * The type of the currency, which can be either 'standard' for standard ISO 4217 currencies or 'custom' for user-defined currencies.
   * This field is not required when creating a custom currency and is read-only after creation.
   * The system will validate that the provided code matches the specified currency type.
   */
  @visibility(Lifecycle.Read)
  currency_type: CurrencyType;
}

/**
 * StandardCurrency represents a standard ISO 4217 currency supported by the billing system.
 * It includes a currency code that adheres to the ISO 4217 standard and is read-only, as standard currencies cannot be modified or deleted by users.
 * The currency_type field is set to 'standard' to differentiate it from custom currencies.
 */
@friendlyName("BillingStandardCurrencyType")
model StandardCurrencyType extends Currency {
  /**
   * The currency code for the standard currency, which should be a valid ISO 4217 code representing the currency (e.g., USD for US Dollar, EUR for Euro).
   * This field is read-only and cannot be modified by users, as standard currencies are predefined and managed by the system.
   * The system will validate that the provided code matches a valid ISO 4217 code for standard currencies.
   */
  @extension("x-go-type", "string")
  code: Shared.CurrencyCode;

  /**
   * The type of the currency, which is 'standard' for standard ISO 4217 currencies.
   * This field is read-only and is set to 'standard' for all instances of StandardCurrency.
   */
  currency_type: CurrencyType.Standard;
}

/**
 * CustomCurrency represents a user-defined currency that can be created by users for billing purposes.
 * It includes a unique currency code that does not conflict with any existing standard ISO 4217 codes, and a creation timestamp to track when the custom currency was created.
 * The currency_type field is set to 'custom' to differentiate it from standard currencies.
 */
@friendlyName("BillingCustomCurrencyType")
model CustomCurrencyType extends Currency {
  /**
   * The currency code for the custom currency, which should be a unique code based on the ISO 4217 format but not conflicting with any existing standard currency codes.
   * This field is required when creating a custom currency and is read-only after creation.
   * The system will validate that the provided code does not match any standard ISO 4217 code and is unique among custom currencies.
   */
  @extension("x-go-type", "string")
  code: CustomCurrencyCode;

  /**
   * The type of the currency, which is 'custom' for user-defined currencies.
   * This field is read-only and is set to 'custom' for all instances of CustomCurrencyType.
   */
  currency_type: CurrencyType.Custom;
}
