import "@typespec/http";
import "../types.tsp";
import "./ratecards.tsp";

namespace OpenMeter.ProductCatalog;

/**
 * Common errors for the product catalog API.
 */
alias CommonSubscriptionErrors =
  | OpenMeter.BadRequestError
  | OpenMeter.UnauthorizedError
  | OpenMeter.ForbiddenError
  | OpenMeter.InternalServerErrorError
  | OpenMeter.ServiceUnavailableError
  | OpenMeter.PreconditionFailedError
  | OpenMeter.Error;
