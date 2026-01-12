/**
 * OAuth2 Authorization Flow as defined in RFC 6749 (https://tools.ietf.org/html/rfc6749#section-4.1).
 *
 * The flow consists of three main steps:
 * 1. Client initiates installation (AppOAuth2InstallResponse)
 * 2. User authorizes the app and is redirected back with authorization code or error
 * 3. Client exchanges authorization code for access token (handled separately)
 */
import "@typespec/http";

using TypeSpec.Http;

namespace Apps;

/**
 * Response from the billing app installation endpoint to initiate the OAuth2 authorization flow.
 *
 * This response contains the authorization URL where the user should be redirected
 * to grant permissions to the application. The URL includes all necessary OAuth2
 * parameters such as client_id, redirect_uri, scope, and state.
 */
@friendlyName("BillingAppOAuth2InstallResponse")
@example(#{
  url: "https://provider.com/oauth/authorize?client_id=xxx&redirect_uri=...&state=random123",
})
model AppOAuth2InstallResponse {
  /**
   * The URL to start the OAuth2 authorization code grant flow.
   */
  @format("uri")
  @visibility(Lifecycle.Read)
  url: string;
}

/**
 * OAuth2 authorization code grant error types as defined in RFC 6749 Section 4.1.2.1.
 *
 * These error codes are returned in the redirect URI when the authorization request fails.
 *
 * @see https://tools.ietf.org/html/rfc6749#section-4.1.2.1
 */
@friendlyName("BillingAppOAuth2AuthorizationCodeGrantErrorType")
enum AppOAuth2AuthorizationCodeGrantErrorType {
  /**
   * The request is missing a required parameter, includes an invalid parameter value,
   * includes a parameter more than once, or is otherwise malformed.
   */
  InvalidRequest: "invalid_request",

  /**
   * The client is not authorized to request an authorization code using this method.
   */
  UnauthorizedClient: "unauthorized_client",

  /**
   * The resource owner or authorization server denied the request.
   */
  AccessDenied: "access_denied",

  /**
   * The authorization server does not support obtaining an authorization code using this method.
   */
  UnsupportedResponseType: "unsupported_response_type",

  /**
   * The requested scope is invalid, unknown, or malformed.
   */
  InvalidScope: "invalid_scope",

  /**
   * The authorization server encountered an unexpected condition that prevented it from fulfilling the request.
   */
  ServerError: "server_error",

  /**
   * The authorization server is currently unable to handle the request due to a temporary overloading or maintenance of the server.
   */
  TemporarilyUnavailable: "temporarily_unavailable",
}

/**
 * OAuth2 authorization code grant callback parameters.
 *
 * When the authorization server redirects back to the client's callback URL,
 * it includes these parameters in the query string. The response can indicate
 * either success (with a 'code' parameter) or failure (with an 'error' parameter).
 *
 * Success and error params are combined into a single model because OAuth2
 * uses the same callback URL for both scenarios, differentiated by the presence
 * of either 'code' or 'error' parameters.
 */
#suppress "@openmeter/api-spec/repeated-prefix-grouping" "OAuth2 spec defines error_description and error_uri as separate parameters"
@friendlyName("OAuth2AuthorizationCodeGrantParams")
model AppOAuth2AuthorizationCodeGrantParams {
  ...AppOAuth2AuthorizationCodeGrantSuccessParams;
  ...AppOAuth2AuthorizationCodeGrantErrorParams;
}

/**
 * OAuth2 authorization code grant success query params.
 *
 * If the resource owner grants the access request, the authorization
 * server issues an authorization code and delivers it to the client by
 * adding the following parameters to the query component of the
 * redirection URI using the "application/x-www-form-urlencoded" format.
 */
@friendlyName("BillingAppOAuth2AuthorizationCodeGrantSuccessParams")
model AppOAuth2AuthorizationCodeGrantSuccessParams {
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
#suppress "@openmeter/api-spec/repeated-prefix-grouping" "OAuth2 spec defines error_description and error_uri as separate parameters"
@friendlyName("BillingAppOAuth2AuthorizationCodeGrantErrorParams")
model AppOAuth2AuthorizationCodeGrantErrorParams {
  /**
   * Error code.
   * Required with the error response.
   */
  @query
  error?: AppOAuth2AuthorizationCodeGrantErrorType;

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
