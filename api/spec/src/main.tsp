import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "@typespec/versioning";

import "./auth.tsp";
import "./errors.tsp";
import "./types.tsp";

import "./app";
import "./query.tsp";
import "./customer.tsp";
import "./events.tsp";
import "./meters.tsp";
import "./portal.tsp";
import "./subjects.tsp";
import "./debug.tsp";
import "./notification";
import "./entitlements";
import "./billing";
import "./productcatalog";

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
@server("https://127.0.0.1", "Local")
@tagMetadata(
  "Meters",
  #{
    description: "Meters are used to aggregate usage events. They are designed to offer multiple aggregation methods and window sizes, making it highly adaptable to your specific needs.",
  }
)
@tagMetadata(
  "Events",
  #{
    description: "A usage event can be anything you need to track accurately over time for billing or analytics purposes. [Read more](https://openmeter.io/docs/concepts/usage-events).",
  }
)
@tagMetadata(
  "Entitlements (Beta)",
  #{
    description: "Entitlements make it possible to implement complex pricing scenarios such as monthly quotas, prepaid billing, and per-customer pricing. Don't just list usage limits on your marketing page—enforce them within your application to control the cost of expensive resources like LLMs.",
  }
)
@tagMetadata(
  "Notification (Beta)",
  #{
    description: "Notifications provide automated triggers when specific entitlement balances and usage thresholds are reached, ensuring that your customers and sales teams are always informed. For example, you can email your customers when they reach 75%, 100%, and 150% of their monthly token allowance and notify your sales team when an overage happens to drive expansions.",
  }
)
@tagMetadata(
  "Portal",
  #{
    description: "With the Consumer Portal, you can build in-app user-facing dashboards where your customers can follow their consumption in real-time.",
  }
)
namespace OpenMeter;
