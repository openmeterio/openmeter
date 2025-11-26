import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "@typespec/versioning";

import "./auth.tsp";
import "./errors.tsp";
import "./types.tsp";
import "./filter.tsp";

import "./app";
import "./query.tsp";
import "./customer";
import "./events.tsp";
import "./meters.tsp";
import "./portal.tsp";
import "./subjects.tsp";
import "./debug.tsp";
import "./notification";
import "./entitlements";
import "./entitlements/v2";
import "./billing";
import "./productcatalog";
import "./info";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

/**
 * OpenMeter is a cloud native usage metering service.
 * The OpenMeter API allows you to ingest events, query meter usage, and manage resources.
 */
@service(#{ title: "OpenMeter API" })
@info(#{
  version: "1.0.0",
  license: #{
    name: "Apache 2.0",
    url: "http://www.apache.org/licenses/LICENSE-2.0.html",
  },
})
@server("https://127.0.0.1", "Local")
// Tags will appear in alphabetical order in the API documentation
@tagMetadata(
  "Apps",
  #{
    description: "Manage integrations for extending OpenMeter's functionality. ",
  }
)
@tagMetadata("App: Stripe", #{ description: "Support for Stripe billing." })
@tagMetadata(
  "App: Custom Invoicing",
  #{ description: "Interface third party invoicing and payment systems." }
)
@tagMetadata(
  "Billing",
  #{ description: "Manage your billing profiles and invoices. " }
)
@tagMetadata(
  "Customers",
  #{
    description: "Manage customer subscription lifecycles and plan assignments. ",
  }
)
@tagMetadata("Debug", #{ description: "Debugging and testing endpoints." })
@tagMetadata(
  "Entitlements",
  #{
    description: "With Entitlements, you can define and enforce usage limits, implement quota-based pricing, and manage access to features in your application.",
  }
)
@tagMetadata(
  "Events",
  #{
    description: "Events are used to track usage of your product or service. Events are processed asynchronously by the meters, so they may not be immediately available for querying.",
  }
)
@tagMetadata(
  "Lookup Information",
  #{ description: "Lookup information for static data like currencies" }
)
@tagMetadata(
  "Notifications",
  #{
    description: "Notifications provide automated triggers when specific entitlement balances and usage thresholds are reached, ensuring that your customers and sales teams are always informed. Notify customers and internal teams when specific conditions are met, like reaching 75%, 100%, and 150% of their monthly usage allowance. [Read more](https://openmeter.io/docs/guides/notifications/overview).",
  }
)
@tagMetadata(
  "Portal",
  #{
    description: "With the Consumer Portal, you can build in-app user-facing dashboards where your users can track their usage in real-time. Subject scoped portal tokens can be generated on your behalf to allow restricted access to the OpenMeter API.",
  }
)
@tagMetadata(
  "Product Catalog",
  #{
    description: "Configure and manage your product plans, pricing tiers, and subscription offerings. ",
  }
)
@tagMetadata(
  "Subjects",
  #{
    description: "⚠️ __Deprecated__: Subjects as managable entities are being depracated, use customers with subject key usage attribution instead. \n\n Subjects are entities that consume resources you wish to meter. These can range from users, servers, and services to devices. The design of subjects is intentionally generic, enabling flexible application across various metering scenarios. Typically, a subject acts as a unique identifier within your system for a user or customer. Meters are aggregating events for each subject.",
  }
)
@tagMetadata(
  "Subscriptions",
  #{
    description: "With Subscriptions, you can easily start, cancel, and manage customer subscriptions. For example, provisioning them on a specific plan or assigning custom rate cards.",
  }
)
namespace OpenMeter;
