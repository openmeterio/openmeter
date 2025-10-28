import "@typespec/http";

using TypeSpec.Http;

namespace OAuth2;

/**
 * Response from the client app (OpenMeter backend) to start the OAuth2 flow.
 *
 */
@friendlyName("ClientAppStartResponse")
model ClientAppStartResponse {
  /**
   * The URL to start the OAuth2 authorization code grant flow.
   */
  url: string;
}

/**
 * OAuth2 authorization code grant error types.
 */
@friendlyName("OAuth2AuthorizationCodeGrantErrorType")
enum AuthorizationCodeGrantErrorType {
  /**
   * The request is missing a required parameter, includes an invalid parameter value,
   * includes a parameter more than once, or is otherwise malformed.
   */
  invalid_request: "invalid_request",

  /**
   * The client is not authorized to request an authorization code using this method.
   */
  unauthorized_client: "unauthorized_client",

  /**
   * The resource owner or authorization server denied the request.
   */
  access_denied: "access_denied",

  /**
   * The authorization server does not support obtaining an authorization code using this method.
   */
  unsupported_response_type: "unsupported_response_type",

  /**
   * The requested scope is invalid, unknown, or malformed.
   */
  invalid_scope: "invalid_scope",

  /**
   * The authorization server encountered an unexpected condition that prevented it from fulfilling the request.
   */
  server_error: "server_error",

  /**
   * The authorization server is currently unable to handle the request due to a temporary overloading or maintenance of the server.
   */
  temporarily_unavailable: "temporarily_unavailable",
}

/**
 * OAuth2 authorization code grant params.
 *
 * We combine the success and error params into a single model.
 * Because of this success's `code` and error's `error` are optional fields.
 */
@friendlyName("OAuth2AuthorizationCodeGrantParams")
model AuthorizationCodeGrantParams {
  ...AuthorizationCodeGrantSuccessParams;
  ...AuthorizationCodeGrantErrorParams;
}

/**
 * OAuth2 authorization code grant success query params.
 *
 * If the resource owner grants the access request, the authorization
 * server issues an authorization code and delivers it to the client by
 * adding the following parameters to the query component of the
 * redirection URI using the "application/x-www-form-urlencoded" format.
 */
@friendlyName("OAuth2AuthorizationCodeGrantSuccessParams")
model AuthorizationCodeGrantSuccessParams {
  /**
   * Required if the "state" parameter was present in the client authorization request.
   * The exact value received from the client:
   *
   * Unique, randomly generated, opaque, and non-guessable string that is sent
   * when starting an authentication request and validated when processing the response.
   */
  @query
  state?: string;

  /**
   * Authorization code which the client will later exchange for an access token.
   * Required with the success response.
   */
  @query
  code?: string;
}

/**
 * OAuth2 authorization code grant error query params.
 *
 * If the request fails due to a missing, invalid, or mismatching
 * redirection URI, or if the client identifier is missing or invalid,
 * the authorization server SHOULD inform the resource owner of the
 * error and MUST NOT automatically redirect the user-agent to the
 * invalid redirection URI.
 */
@friendlyName("OAuth2AuthorizationCodeGrantErrorParams")
model AuthorizationCodeGrantErrorParams {
  /**
   * Error code.
   * Required with the error response.
   */
  @query
  error?: AuthorizationCodeGrantErrorType;

  /**
   * Optional human-readable text providing additional information,
   * used to assist the client developer in understanding the error that occurred.
   */
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  @query
  error_description?: string;

  /**
   * Optional uri identifying a human-readable web page with
   * information about the error, used to provide the client
   * developer with additional information about the error
   */
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  @query
  error_uri?: string;
}
