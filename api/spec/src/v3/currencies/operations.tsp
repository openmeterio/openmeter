import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./currency.tsp";
import "./custom.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Currencies;

/**
 * Response model for listing currencies.
 * It contains a list of currencies supported by the billing system.
 * Each currency in the list includes details such as its name, symbol, code, and whether it is a custom currency or a standard currency. This response is used when clients request to retrieve all supported currencies from the billing system.
 */
@friendlyName("CurrenciesResponse")
model CurrenciesResponse {
  @Http.statusCode _: 200;

  @pageItems
  data: Currency[];
}

interface CurrenciesOperations {
  /**
   * List all currencies supported by the billing system.
   * This includes both standard currencies provided by the system and custom currencies created by users.
   * The response will contain details for each currency, such as its name, symbol, code, and whether it is a custom currency or a standard currency.
   */
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @get
  @operationId("list-currencies")
  @summary("List currencies")
  list(): CurrenciesResponse | Common.ErrorResponses;

  /**
   * Create a custom currency. This operation allows users to define their own custom currency for billing purposes.
   * The request body should include the details of the custom currency, such as its name, symbol, and a unique code that follows the ISO 4217 format but does not conflict with any existing standard currency codes.
   * The response will include the details of the created custom currency, including its unique identifier, name, symbol, code, and creation timestamp.
   * This operation is only applicable for creating custom currencies; standard currencies are predefined by the system and cannot be created or modified by users.
   */
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @post
  @operationId("create-custom-currency")
  @summary("Create custom currency")
  create(
    @header contentType: "application/json",
    @body body: Currencies.CustomCurrency,
  ): Shared.CreateResponse<CustomCurrency> | Common.ErrorResponses;
}
