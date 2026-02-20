import "@typespec/openapi3";
import "../shared/index.tsp";

using TypeSpec.OpenAPI;

namespace Currencies;

@minLength(3)
@maxLength(24)
scalar CustomCurrencyCode extends string;

@friendlyName("BillingCustomCurrencyBase")
model CurrencyBase {
  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * Custom Currency name
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  @minLength(1)
  @maxLength(256)
  name: string;

  /**
   * Symbol representation of the currency.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  symbol: string;
}
