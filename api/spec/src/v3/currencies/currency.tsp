import "@typespec/openapi3";
import "../shared/index.tsp";

using TypeSpec.OpenAPI;

namespace Currencies;

/**
 * Currency type for custom currencies.
 * It should be a unique code but not conflicting with any existing standard currency codes.
 */
@friendlyName("BillingCurrencyType")
enum CurrencyType {
  Fiat: "fiat",
  Custom: "custom",
}

/**
 * Fiat or custom currency.
 */
@friendlyName("BillingCurrency")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union Currency {
  @summary("Fiat currency")
  fiat: CurrencyFiat,

  @summary("Custom currency")
  custom: CurrencyCustom,
}

/**
 * Describes the common properties of a currency. It is used as a base model for both standard currencies and custom currencies.
 * It includes properties such as id, name, and symbol that are shared by both types of currencies.
 */
@friendlyName("BillingCustomCurrencyBase")
model CurrencyBase<T extends CurrencyType> {
  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * The type of the currency.
   */
  @visibility(Lifecycle.Read)
  type: T;

  /**
   * The name of the currency. It should be a human-readable string that represents the name of the currency, such as "US Dollar" or "Euro".
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  @minLength(1)
  @maxLength(256)
  name: string;

  /**
   * The symbol of the currency. It should be a string that represents the symbol of the currency, such as "$" for US Dollar or "€" for Euro.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  @minLength(1)
  symbol?: string;
}

/**
 * Currency describes a currency supported by the billing system.
 */
@friendlyName("BillingCurrencyFiat")
model CurrencyFiat {
  ...CurrencyBase<CurrencyType.Fiat>;

  /**
   * The currency code. For standard currencies, it should be a standard ISO 4217 code representing the currency.
   * For custom currencies, it should be a unique code but not conflicting with any existing standard currency codes.
   */
  @visibility(Lifecycle.Read)
  code: Shared.CurrencyCode;
}

/**
 * Custom currency code.
 * It should be a unique code but not conflicting with any existing standard currency codes.
 */
@friendlyName("BillingCurrencyCodeCustom")
@minLength(3)
@maxLength(24)
scalar CurrencyCodeCustom extends string;

/**
 * Describes custom currency which can be created by users.
 * Custom currencies are identified by a unique code that follows the ISO 4217 format but does not conflict with any existing standard currency codes.
 * Each custom currency has a name, an optional symbol, and a creation timestamp.
 * Custom currencies allow users to define their own currencies for billing purposes, enabling flexibility in cost management and reporting.
 */
@friendlyName("BillingCurrencyCustom")
model CurrencyCustom {
  ...CurrencyBase<CurrencyType.Custom>;

  /**
   * The currency code for the custom currency. It should be a unique code but not conflicting with any existing standard currency codes.
   * This field is required when creating a custom currency and is read-only after creation.
   * The system will validate that the provided code does not match any standard ISO 4217 code and is unique among custom currencies.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  code: CurrencyCodeCustom;

  /**
   * An ISO-8601 timestamp representation of the custom currency creation date.
   */
  @visibility(Lifecycle.Read)
  created_at?: Shared.DateTime;
}
