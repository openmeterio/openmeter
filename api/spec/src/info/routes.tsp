import "../rest.tsp";

namespace OpenMeter.Info;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@route("/v1/info/currencies")
@tag("Lookup Information")
interface Currencies {
  /**
   * List all supported currencies.
   */
  @get
  @operationId("listCurrencies")
  @summary("List supported currencies")
  listCurrencies(): Currency[] | OpenMeter.CommonErrors;
}
