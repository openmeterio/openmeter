import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

using TypeSpec.OpenAPI;
using TypeSpec.Http;

namespace Common;

/**
 * Bad Request.
 */
@useRef("../../../../common/definitions/errors.yaml#/components/responses/BadRequest")
@friendlyName("BadRequest")
model BadRequest {
  @statusCode _: 400;
}

/**
 * Unauthorized.
 */
@useRef("../../../../common/definitions/errors.yaml#/components/responses/Unauthorized")
@friendlyName("Unauthorized")
model Unauthorized {
  @statusCode _: 401;
}

/**
 * Forbidden.
 */
@useRef("../../../../common/definitions/errors.yaml#/components/responses/Forbidden")
@friendlyName("Forbidden")
model Forbidden {
  @statusCode _: 403;
}

/**
 * Not Found.
 */
@useRef("../../../../common/definitions/errors.yaml#/components/responses/NotFound")
@friendlyName("NotFound")
model NotFound {
  @statusCode _: 404;
}

/**
 * Conflict.
 */
@useRef("../../../../common/definitions/errors.yaml#/components/responses/Conflict")
@friendlyName("Conflict")
model Conflict {
  @statusCode _: 409;
}

/**
 * Gone.
 */
@useRef("../../../../common/definitions/errors.yaml#/components/responses/Gone")
@friendlyName("Gone")
model Gone {
  @statusCode _: 410;
}

/**
 * Unsupported Media Type.
 */
@friendlyName("UnsupportedMediaType")
@useRef("../../../../common/definitions/errors.yaml#/components/responses/UnsupportedMediaType")
@friendlyName("UnsupportedMediaType")
model UnsupportedMediaType {
  @statusCode _: 415;
}

/**
 * Unprocessable Content.
 */
@friendlyName("UnprocessableContent")
@useRef("../../../../common/definitions/errors.yaml#/components/responses/UnprocessableContent")
@friendlyName("UnprocessableContent")
model UnprocessableContent {
  @statusCode _: 422;
}

/**
 * Too Many Requests.
 */
@friendlyName("TooManyRequests")
@useRef("../../../../common/definitions/errors.yaml#/components/responses/TooManyRequests")
@friendlyName("TooManyRequests")
model TooManyRequests {
  @statusCode _: 429;
}

/**
 * Internal Server Error.
 */
@useRef("../../../../common/definitions/errors.yaml#/components/responses/Internal")
@friendlyName("Internal")
model Internal {
  @statusCode _: 500;
}

/**
 * Not Implemented.
 */
@friendlyName("NotImplemented")
@useRef("../../../../common/definitions/errors.yaml#/components/responses/NotImplemented")
@friendlyName("NotImplemented")
model NotImplemented {
  @statusCode _: 501;
}

/**
 * Not Available.
 */
@useRef("../../../../common/definitions/errors.yaml#/components/responses/NotAvailable")
@friendlyName("NotAvailable")
model NotAvailable {
  @statusCode _: 503;
}

alias ErrorResponses = BadRequest | Unauthorized | Forbidden;
