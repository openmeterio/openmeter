import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";

import "..";

import "./auth.tsp";
import "./subjects.tsp";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

/**
 * OpenMeter is a cloud native usage metering service.
 * The OpenMeter API allows you to ingest events, query meter usage, and manage resources.
 */
@service({
  title: "OpenMeter Cloud API",
})
@info({
  version: "1.0.0",
  license: {
    name: "Apache 2.0",
    url: "http://www.apache.org/licenses/LICENSE-2.0.html",
  },
})
@server("https://127.0.0.1:8888", "Local")
@server("https://openmeter.cloud", "Cloud")
@useAuth(NoAuth | CloudTokenAuth | CloudCookieAuth)
namespace OpenMeterCloud;

@route("/api/v1/events")
@tag("Events")
interface Events extends OpenMeter.Events {}

@route("/api/v1/meters")
@tag("Meters")
interface Meters extends OpenMeter.Meters {
  /**
   * Create a meter.
   */
  @post
  @operationId("createMeter")
  create(@body meter: OpenMeter.Meter): {
    @statusCode _: 201;
    @body body: OpenMeter.Meter;
  } | OpenMeter.CommonErrors;

  /**
   * Delete a meter.
   */
  @delete
  @operationId("deleteMeter")
  delete(@path meterIdOrSlug: OpenMeter.MeterIdentifier): void | OpenMeter.CommonErrors;
}

namespace OpenMeterCloud.Portal {
  @route("/api/v1/portal/tokens")
  @tag("Portal")
  interface Tokens extends OpenMeter.Portal.Tokens {
    /**
     * List tokens.
     */
    @get
    @operationId("listPortalTokens")
    list(
      @query
      @minValue(1)
      @maxValue(100)
      limit?: integer = 25,
    ): OpenMeter.Portal.Token[] | OpenMeter.CommonErrors;

    /**
     * Invalidates consumer portal tokens by ID or subject.
     */
    @route("/invalidate")
    @post
    @operationId("invalidatePortalTokens")
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

  @route("/api/v1/portal/meters")
  @tag("Portal")
  @useAuth(PortalTokenAuth | CloudPortalTokenAuth)
  interface Meters extends OpenMeter.Portal.Meters {}
}

@route("/api/v1/debug")
@tag("Debug")
interface Debug extends OpenMeter.Debug {}

namespace OpenMeterCloud.App {
  @route("/api/v1/marketplace")
  @tag("Apps")
  interface Marketplace extends OpenMeter.App.Marketplace {}

  @route("/api/v1/apps")
  @tag("Apps")
  interface Apps extends OpenMeter.App.Apps {}

  @tag("Apps")
  interface AppStripe extends OpenMeter.App.AppStripe {}
}

namespace OpenMeterCloud.Notification {
  @route("/api/v1/notification/channels")
  @tag("Notification (Experimental)")
  interface Channels extends OpenMeter.Notification.Channels {}

  @route("/api/v1/notification/rules")
  @tag("Notification (Experimental)")
  interface Rules extends OpenMeter.Notification.Rules {}

  @route("/api/v1/notification/events")
  @tag("Notification (Experimental)")
  interface Events extends OpenMeter.Notification.Events {}

  @route("/api/v1/notification/webhook/svix")
  @tag("Notification (Experimental)")
  interface SvixOperationalWebhooks extends OpenMeter.Notification.SvixOperationalWebhooks {}
}

namespace OpenMeterCloud.Entitlements {
  @route("/api/v1/entitlements")
  @tag("Entitlements (Experimental)")
  interface Entitlements extends OpenMeter.Entitlements.Entitlements {}

  @route("/api/v1/features")
  @tag("Entitlements (Experimental)")
  interface Features extends OpenMeter.Entitlements.Features {}

  @route("/api/v1/grants")
  @tag("Entitlements (Experimental)")
  interface Grants extends OpenMeter.Entitlements.Grants {}

  @route("/api/v1/subjects/{subjectIdOrKey}/entitlements")
  @tag("Entitlements (Experimental)")
  interface Subjects extends OpenMeter.Entitlements.Subjects {}
}

@route("/api/v1/customers")
@tag("Customers")
interface Customers extends OpenMeter.Customers {}
