import "@typespec/openapi3";
import "../shared/index.tsp";

using TypeSpec.OpenAPI;

namespace Currencies;

@minLength(3)
@maxLength(24)
scalar CustomCurrencyCode extends string;

/**
 * CurrencyBase describes the common properties of a currency. It is used as a base model for both standard currencies and custom currencies.
 * It includes properties such as id, name, and symbol that are shared by both types of currencies.
 */
@friendlyName("BillingCustomCurrencyBase")
model CurrencyBase {
  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

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
