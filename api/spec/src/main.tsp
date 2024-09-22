import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "@typespec/versioning";

import "./auth.tsp";
import "./errors.tsp";
import "./types.tsp";

import "./query.tsp";
import "./customer.tsp";
import "./events.tsp";
import "./meters.tsp";
import "./portal.tsp";
import "./debug.tsp";
import "./notification.tsp";
import "./entitlements";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;
using TypeSpec.Versioning;

/**
 * OpenMeter is a cloud native usage metering service.
 * The OpenMeter API allows you to ingest events, query meter usage, and manage resources.
 */
@service({
  title: "OpenMeter API",
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
namespace OpenMeter;
