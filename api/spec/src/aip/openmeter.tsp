import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "./shared/index.tsp";
import "./events/index.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

/**
 * OpenMeter is a cloud native usage metering and billing service.
 * The OpenMeter API allows you to ingest events, query meter usage, and manage resources.
 */
@service(#{ title: "OpenMeter API" })
@info(#{
  version: "1.0.0",
  license: #{
    name: "Apache 2.0",
    url: "http://www.apache.org/licenses/LICENSE-2.0.html",
  },
  termsOfService: "https://openmeter.cloud/terms-of-service",
})
@server("https://127.0.0.1/api/v3", "Local")
@server("https://openmeter.cloud/api/v3", "Cloud")
@tagMetadata(Shared.MetersTag, #{ description: Shared.MetersDescription })
@tagMetadata(
  Shared.MeteringEventsTag,
  #{ description: Shared.MeteringEventsDescription }
)
namespace OpenMeter;

@route("/metering-events")
@tag(Shared.MeteringEventsTag)
@friendlyName("OpenMeter: ${Shared.MeteringEventsTag}")
interface MeteringEventsEndpoints extends Events.MeteringEventsOperations {}
