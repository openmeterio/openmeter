import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.Portal;

@route("/api/v1/portal/tokens")
@tag("Portal")
@friendlyName("PortalTokens")
interface TokensEndpoints {
  /**
   * Create a consumer portal token.
   */
  @post
  @operationId("createPortalToken")
  @summary("Create consumer portal token")
  create(@body token: Token): {
    // TODO: Should be 201
    @statusCode _: 200;

    @body body: Token;
  } | CommonErrors;

  /**
   * List tokens.
   */
  @get
  @operationId("listPortalTokens")
  @summary("List consumer portal tokens")
  list(
    @query
    @minValue(1)
    @maxValue(100)
    limit?: integer = 25,
  ): Token[] | CommonErrors;

  /**
   * Invalidates consumer portal tokens by ID or subject.
   */
  @route("/invalidate")
  @post
  @operationId("invalidatePortalTokens")
  @summary("Invalidate portal tokens")
  invalidate(
    /**
     * Invalidate a portal token by ID.
     */
    id?: string,

    /**
     * Invalidate all portal tokens for a subject.
     */
    subject?: string,
  ): void | OpenMeter.CommonErrors;
}

/**
 * A consumer portal token.
 *
 * Validator doesn't obey required for readOnly properties
 * See: https://github.com/stoplightio/spectral/issues/1274
 */
@friendlyName("PortalToken")
model Token {
  @visibility(Lifecycle.Read)
  @example("01G65Z755AFWAKHE12NY0CQ9FH")
  id?: ULID;

  @example("customer-1")
  subject: string;

  @visibility(Lifecycle.Read)
  expiresAt?: DateTime;

  @visibility(Lifecycle.Read)
  expired?: boolean;

  @visibility(Lifecycle.Read)
  createdAt?: DateTime;

  // TODO: technically the visibility is incorrect, because the token is only returned at creation.

  /**
   * The token is only returned at creation.
   */
  @visibility(Lifecycle.Read)
  @example("om_portal_IAnD3PpWW2A2Wr8m9jfzeHlGX8xmCXwG.y5q4S-AWqFu6qjfaFz0zQq4Ez28RsnyVwJffX5qxMvo")
  token?: string;

  // TODO: make sure items are unique

  /**
   * Optional, if defined only the specified meters will be allowed.
   */
  @example(#["tokens_total"])
  allowedMeterSlugs?: string[];
}

@route("/api/v1/portal/meters")
@tag("Portal")
@useAuth(PortalTokenAuth)
@friendlyName("PortalMeters")
interface MetersEndpoints {
  /**
   * Query meter for consumer portal. This endpoint is publicly exposable to consumers.
   */
  @get
  @route("/{meterSlug}/query")
  @operationId("queryPortalMeter")
  @summary("Query meter")
  @sharedRoute
  queryJson(@path meterSlug: Key, ...OmitProperties<MeterQuery, "subject">): {
    @header contentType: "application/json";
    @body _: MeterQueryResult;
  } | NotFoundError | CommonErrors;

  /**
   * Query meter for consumer portal. This endpoint is publicly exposable to consumers.
   */
  @get
  @route("/{meterSlug}/query")
  @operationId("queryPortalMeter")
  @summary("Query meter")
  @sharedRoute
  queryCsv(@path meterSlug: Key, ...OmitProperties<MeterQuery, "subject">): {
    @header contentType: "text/csv";

    @body
    @example("""
      window_start,window_end,subject,model,type,value
      2023-01-01T00:00:00Z,2023-01-01T00:01:00Z,customer_1,gpt-4-turbo,input,12
      2023-01-01T00:01:00Z,2023-01-02T00:02:00Z,customer_1,gpt-4-turbo,input,20
      2023-01-01T00:02:00Z,2023-01-02T00:03:00Z,customer_2,gpt-4-turbo,output,4
      """)
    _: string;
  } | NotFoundError | CommonErrors;
}
