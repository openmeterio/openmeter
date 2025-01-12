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
// Tags will appear in reverse order in the API documentation
@tagMetadata(
  "Apps (Private Preview)",
  #{
    description: "Manage integrations for extending OpenMeter's functionality. Currently in private preview - [contact us](mailto:contact@openmeter.cloud) for access.",
  }
)
@tagMetadata(
  "Billing (Private Preview)",
  #{
    description: "Manage your billing profiles and invoices. Currently in private preview - [contact us](mailto:contact@openmeter.cloud) for access.",
  }
)
@tagMetadata(
  "Customers (Private Preview)",
  #{
    description: "Manage customer subscription lifecycles and plan assignments. Currently in private preview - [contact us](mailto:contact@openmeter.cloud) for access.",
  }
)
@tagMetadata(
  "Subscriptions (Private Preview)",
  #{
    description: "With Subscriptions, you can manage your customers' subscriptions. Request access by [contacting us](mailto:contact@openmeter.cloud).",
  }
)
@tagMetadata(
  "Product Catalog (Private Preview)",
  #{
    description: "Configure and manage your product plans, pricing tiers, and subscription offerings. Currently in private preview - [contact us](mailto:contact@openmeter.cloud) for access.",
  }
)
@tagMetadata(
  "Notifications (Beta)",
  #{
    description: "Notifications provide automated triggers when specific entitlement balances and usage thresholds are reached, ensuring that your customers and sales teams are always informed. Notify customers and internal teams when specific conditions are met, like reaching 75%, 100%, and 150% of their monthly usage allowance. [Read more](https://openmeter.io/docs/guides/notifications/overview).",
  }
)
@tagMetadata(
  "Entitlements (Beta)",
  #{
    description: "With Entitlements, you can define and enforce usage limits, implement quota-based pricing, and manage access to features in your application.",
  }
)
@tagMetadata("Debug", #{ description: "Debugging and testing endpoints." })
@tagMetadata(
  "Portal",
  #{
    description: "With the Consumer Portal, you can build in-app user-facing dashboards where your users can track their usage in real-time. Subject scoped portal tokens can be generated on your behalf to allow restricted access to the OpenMeter API.",
  }
)
@tagMetadata(
  "Subjects",
  #{
    description: "Subjects are entities that consume resources you wish to meter. These can range from users, servers, and services to devices. The design of subjects is intentionally generic, enabling flexible application across various metering scenarios. Typically, a subject acts as a unique identifier within your system for a user or customer. Meters are aggregating events for each subject.",
  }
)
@tagMetadata(
  "Events",
  #{
    description: "Events are used to track usage of your product or service. Events are processed asynchronously by the meters, so they may not be immediately available for querying.",
  }
)
@tagMetadata(
  "Meters",
  #{
    description: "Meters specify how to aggregate events for billing and analytics purposes. Meters can be configured with multiple aggregation methods and groupings. Multiple meters can be created for the same event type, enabling flexible metering scenarios.",
  }
)
namespace OpenMeter;
