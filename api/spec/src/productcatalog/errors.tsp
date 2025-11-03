import "@typespec/http";
import "../types.tsp";
import "./ratecards.tsp";

namespace OpenMeter.ProductCatalog;

/**
 * Common errors for the product catalog API.
 */
alias CommonSubscriptionErrors =
  | SubscriptionBadRequestErrorResponse
  | SubscriptionConflictErrorResponse
  | OpenMeter.UnauthorizedError
  | OpenMeter.ForbiddenError
  | OpenMeter.InternalServerErrorError
  | OpenMeter.ServiceUnavailableError
  | OpenMeter.PreconditionFailedError
  | OpenMeter.Error;

/**
 * Error extensions for the Subscription Errors.
 */
@friendlyName("SubscriptionErrorExtensions")
union SubscriptionErrorExtensions {
  generic: {
    validationErrors: ErrorExtension[],
  },
}

/**
 * The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing). Variants with ErrorExtensions specific to subscriptions.
 */
@error
@friendlyName("SubscriptionBadRequestErrorResponse")
model SubscriptionBadRequestErrorResponse {
  ...OmitProperties<OpenMeter.BadRequestError, "extensions">;

  /**
   * Additional properties specific to the problem type may be present.
   */
  extensions?: SubscriptionErrorExtensions;
}

/**
 * The request could not be completed due to a conflict with the current state of the target resource.
 * Variants with ErrorExtensions specific to subscriptions.
 */
@error
@friendlyName("SubscriptionConflictErrorResponse")
model SubscriptionConflictErrorResponse {
  ...OmitProperties<OpenMeter.ConflictError, "extensions">;

  /**
   * Additional properties specific to the problem type may be present.
   */
  extensions?: SubscriptionErrorExtensions;
}
