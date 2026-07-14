import "@typespec/openapi3";
import "../shared/index.tsp";

namespace Currencies;

using TypeSpec.OpenAPI;

/**
 * Currency type for custom currencies. It should be a unique code but not
 * conflicting with any existing standard currency codes.
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
 * Fiat or custom currency code.
 */
@friendlyName("BillingCurrencyCode")
@extension("x-go-type", "string")
union CurrencyCode {
  Shared.CurrencyCode,
  // CurrencyCodeCustom,
}

/**
 * Describes the common properties of a currency. It is used as a base model for both standard currencies and custom currencies.
 * It includes properties such as id, name, and symbol that are shared by both types of currencies.
 */
@friendlyName("BillingCustomCurrencyBase")
model CurrencyBase<T extends CurrencyType> {
  /**
   * The type of the currency.
   */
  @visibility(Lifecycle.Read)
  type: T;

  /**
   * The name of the currency. It should be a human-readable string that represents
   * the name of the currency, such as "US Dollar" or "Euro".
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  @minLength(1)
  @maxLength(256)
  name: string;

  /**
   * The symbol of the currency. It should be a string that represents the symbol of
   * the currency, such as "$" for US Dollar or "€" for Euro.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  @minLength(1)
  symbol?: string;

  /**
   * The precision of the currency. It should be a number that represents the number
   * of decimal places used for the currency, such as 2 for US Dollar or Euro.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  precision: uint32;

  /**
   * The decimal mark for the currency. It should be a string that represents the
   * decimal mark of the currency, such as "." for US Dollar or "," for Euro.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  @minLength(1)
  @maxLength(1)
  decimal_mark: string;

  /**
   * The thousand separator for the currency. It should be a string that represents
   * the thousand separator of the currency, such as "," for US Dollar or "." for
   * Euro.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  @minLength(1)
  @maxLength(1)
  thousand_separator: string;
}

/**
 * Currency describes a currency supported by the billing system.
 */
@friendlyName("BillingCurrencyFiat")
model CurrencyFiat {
  ...CurrencyBase<CurrencyType.Fiat>;

  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  @visibility(Lifecycle.Read)
  code: Shared.CurrencyCode;
}

/**
 * Custom currency code. It should be a unique code but not conflicting with any
 * existing fiat currency codes.
 */
@friendlyName("BillingCurrencyCodeCustom")
@minLength(4)
@maxLength(24)
scalar CurrencyCodeCustom extends string;

/**
 * Describes custom currency.
 */
@friendlyName("BillingCurrencyCustom")
model CurrencyCustom {
  ...CurrencyBase<CurrencyType.Custom>;

  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  @visibility(Lifecycle.Create, Lifecycle.Read)
  code: CurrencyCodeCustom;

  /**
   * An ISO-8601 timestamp representation of the custom currency creation date.
   */
  @visibility(Lifecycle.Read)
  created_at: Shared.DateTime;

  /**
   * The list of active cost basis for the custom currency.
   */
  @visibility(Lifecycle.Read)
  cost_basis?: CostBasis[];
}
