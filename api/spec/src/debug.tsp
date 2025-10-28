import "@typespec/http";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;

@route("/api/v1/debug")
@tag("Debug")
@friendlyName("Debug")
interface DebugEndpoints {
  /**
   * Returns debug metrics (in OpenMetrics format) like the number of ingested events since mindnight UTC.
   *
   * The OpenMetrics Counter(s) reset every day at midnight UTC.
   */
  @get
  @route("/metrics")
  @operationId("getDebugMetrics")
  @summary("Get event metrics")
  metrics(): {
    @header contentType: "text/plain";
    @statusCode _: 200;

    @body
    @example("""
      # HELP openmeter_events Number of ingested events
      # TYPE openmeter_events counter
      openmeter_events_total{subject="customer-1"} 12345.0
      openmeter_events_total{subject="customer-1",error="true"} 1.0
      """)
    body: string;
  } | CommonErrors;
}
