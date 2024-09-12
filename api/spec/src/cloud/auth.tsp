using TypeSpec.Http;

/**
 * Cloud API token.
 */
@friendlyName("CloudTokenAuth")
model CloudTokenAuth is BearerAuth {}

/**
 * Cloud API web app cookie.
 */
@friendlyName("CloudCookieAuth")
model CloudCookieAuth is ApiKeyAuth<ApiKeyLocation.cookie, "__session"> {}

/**
 * Cloud consumer portal token.
 */
@friendlyName("CloudPortalTokenAuth")
model CloudPortalTokenAuth is BearerAuth {}