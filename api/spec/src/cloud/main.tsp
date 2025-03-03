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
  termsOfService: "https://openmeter.cloud/terms-of-service",
})
@server("https://127.0.0.1:8888", "Local")
@server("https://openmeter.cloud", "Cloud")
@useAuth(CloudTokenAuth | CloudCookieAuth)
// Tags will appear in reverse order in the API documentation
@tagMetadata(
  "Lookup Information",
  #{ description: "Lookup information for static data like currencies" }
)
@tagMetadata(
  "Apps (Beta)",
  #{
    description: "Manage integrations for extending OpenMeter's functionality. ",
  }
)
@tagMetadata(
  "Billing (Beta)",
  #{ description: "Manage your billing profiles and invoices. " }
)
@tagMetadata(
  "Customers (Beta)",
  #{
    description: "Manage customer subscription lifecycles and plan assignments. ",
  }
)
@tagMetadata(
  "Subscriptions (Beta)",
  #{
    description: "With Subscriptions, you can manage your customers' subscriptions. Request access by [contacting us](mailto:contact@openmeter.cloud).",
  }
)
@tagMetadata(
  "Product Catalog (Beta)",
  #{
    description: "Configure and manage your product plans, pricing tiers, and subscription offerings. ",
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
  @summary("Create meter")
  create(
    @body
    meter: MeterCreate,
  ): {
    @statusCode _: 201;
    @body body: OpenMeter.Meter;
  } | OpenMeter.CommonErrors;

  /**
   * Delete a meter.
   */
  @delete
  @operationId("deleteMeter")
  @summary("Delete meter")
  delete(
    @path meterIdOrSlug: OpenMeter.MeterIdentifier,
  ): void | OpenMeter.CommonErrors;
}

namespace OpenMeterCloud.Portal {
  @route("/api/v1/portal/tokens")
  @tag("Portal")
  interface Tokens extends OpenMeter.Portal.Tokens {}

  @route("/api/v1/portal/meters")
  @tag("Portal")
  @useAuth(CloudPortalTokenAuth)
  interface Meters extends OpenMeter.Portal.Meters {}
}

@route("/api/v1/debug")
@tag("Debug")
interface Debug extends OpenMeter.Debug {}

namespace OpenMeterCloud.App {
  @route("/api/v1/marketplace")
  @tag("Apps (Beta)")
  interface Marketplace extends OpenMeter.App.Marketplace {}

  @route("/api/v1/apps")
  @tag("Apps (Beta)")
  interface Apps extends OpenMeter.App.Apps {}

  @tag("Apps (Beta)")
  interface AppStripe extends OpenMeter.App.AppStripe {}
}

namespace OpenMeterCloud.Notification {
  @route("/api/v1/notification/channels")
  @tag("Notifications (Beta)")
  interface Channels extends OpenMeter.Notification.Channels {}

  @route("/api/v1/notification/rules")
  @tag("Notifications (Beta)")
  interface Rules extends OpenMeter.Notification.Rules {}

  @route("/api/v1/notification/events")
  @tag("Notifications (Beta)")
  interface Events extends OpenMeter.Notification.Events {}

  @route("/api/v1/notification/webhook/svix")
  @tag("Notifications (Beta)")
  interface SvixOperationalWebhooks
    extends OpenMeter.Notification.SvixOperationalWebhooks {}
}

namespace OpenMeterCloud.Entitlements {
  @route("/api/v1/entitlements")
  @tag("Entitlements (Beta)")
  interface Entitlements extends OpenMeter.Entitlements.Entitlements {}

  @route("/api/v1/features")
  @tag("Entitlements (Beta)")
  interface Features extends OpenMeter.Entitlements.Features {}

  @route("/api/v1/grants")
  @tag("Entitlements (Beta)")
  interface Grants extends OpenMeter.Entitlements.Grants {}

  @route("/api/v1/subjects/{subjectIdOrKey}/entitlements")
  @tag("Entitlements (Beta)")
  interface Subjects extends OpenMeter.Entitlements.Subjects {}
}

namespace OpenMeterCloud.Billing {
  @route("/api/v1/customers")
  @tag("Customers (Beta)")
  interface Customers extends OpenMeter.Customers {}

  @route("/api/v1/billing/customers")
  @tag("Billing (Beta)")
  interface CustomerOverrides extends OpenMeter.Billing.CustomerOverrides {}

  @route("/api/v1/billing/profiles")
  @tag("Billing (Beta)")
  interface BillingProfiles extends OpenMeter.Billing.Profiles {}

  @route("/api/v1/billing/invoices")
  @tag("Billing (Beta)")
  interface InvoicesEndpoints extends OpenMeter.Billing.InvoicesEndpoints {}

  @route("/api/v1/billing/invoices/{invoiceId}")
  @tag("Billing (Beta)")
  interface InvoiceEndpoints extends OpenMeter.Billing.InvoiceEndpoints {}

  @route("/api/v1/billing/customers/{customerId}/invoices/simulate")
  @tag("Billing (Beta)")
  interface InvoiceSimulationEndpoints
    extends OpenMeter.Billing.InvoiceSimulationEndpoints {}
}

namespace OpenMeterCloud.ProductCatalog {
  @route("/api/v1/plans")
  @tag("Product Catalog (Beta)")
  interface PlansById extends OpenMeter.ProductCatalog.Plans {}

  @route("/api/v1/subscriptions")
  @tag("Subscriptions (Beta)")
  interface Subscriptions extends OpenMeter.ProductCatalog.Subscriptions {}
}

namespace OpenMeterCloud.Info {
  @route("/api/v1/info/currencies")
  @tag("Lookup Information")
  interface Currencies extends OpenMeter.Info.Currencies {}
}

/**
 * A meter is a configuration that defines how to match and aggregate events.
 */
@friendlyName("MeterCreate")
@example(#{
  slug: "tokens_total",
  description: "AI Token Usage",
  aggregation: OpenMeter.MeterAggregation.SUM,
  windowSize: OpenMeter.WindowSize.MINUTE,
  eventType: "prompt",
  valueProperty: "$.tokens",
  groupBy: #{ `model`: "$.model", type: "$.type" },
})
model MeterCreate is Rest.Resource.ResourceCreateModel<OpenMeter.Meter>;
