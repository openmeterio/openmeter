import "../rest.tsp";

namespace OpenMeter;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@route("/api/v1/info/currencies")
@tag("Lookup Information")
@friendlyName("Currencies")
interface CurrenciesEndpoints {
  /**
   * List all supported currencies.
   */
  @get
  @operationId("listCurrencies")
  @summary("List supported currencies")
  listCurrencies(): Currency[] | CommonErrors;
}
