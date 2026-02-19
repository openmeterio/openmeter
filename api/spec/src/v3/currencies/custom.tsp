import "../shared/index.tsp";

namespace Currencies;

/**
 * Describes custom currency which can be created by users.
 * Custom currencies are identified by a unique code that follows the ISO 4217 format but does not conflict with any existing standard currency codes.
 * Each custom currency has a name, an optional symbol, and a creation timestamp.
 * Custom currencies allow users to define their own currencies for billing purposes, enabling flexibility in cost management and reporting.
 */
@friendlyName("BillingCustomCurrency")
model CustomCurrency {
  ...CurrencyBase;

  /**
   * The currency code for the custom currency. It should be a unique code based on the ISO 4217 format but not conflicting with any existing standard currency codes. This field is required when creating a custom currency and is read-only after creation.
   * The system will validate that the provided code does not match any standard ISO 4217 code and is unique among custom currencies.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  code: CustomCurrencyCode;

  /**
   * An ISO-8601 timestamp representation of the custom currency creation date.
   */
  @visibility(Lifecycle.Read)
  created_at?: Shared.DateTime;
}
