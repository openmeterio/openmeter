import "../rest.tsp";

namespace OpenMeter.Static;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@route("/api/v1/static")
@tag("Static Data")
interface Currencies {
  /**
   * List all supported currencies.
   */
  @get
  @operationId("listCurrencies")
  @route("/currencies")
  @summary("List supported currencies")
  listCurrencies(): Currency[] | OpenMeter.CommonErrors;
}
