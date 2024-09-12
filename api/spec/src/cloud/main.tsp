import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";

import "..";

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
@server(
  "https://127.0.0.1:{port}",
  "Local",
  {
    port: string = "8888",
  }
)
@server("https://openmeter.cloud", "Cloud")
namespace OpenMeterCloud;

@route("/api/v1/events")
@tag("Events")
interface Events extends OpenMeter.Events{}

@route("/api/v1/meters")
@tag("Meters")
interface Meters extends OpenMeter.Meters{
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