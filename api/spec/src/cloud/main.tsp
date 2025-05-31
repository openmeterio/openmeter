import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";

import "..";

import "./auth.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

/**
 * OpenMeter is a cloud native usage metering service.
 * The OpenMeter API allows you to ingest events, query meter usage, and manage resources.
 */
@service(#{ title: "OpenMeter Cloud API" })
@info(#{
  version: "1.0.0",
  license: #{
    name: "Apache 2.0",
    url: "http://www.apache.org/licenses/LICENSE-2.0.html",
  },
  termsOfService: "https://openmeter.cloud/terms-of-service",
})
@server("https://127.0.0.1:8888", "Local")
@server("https://openmeter.cloud", "Cloud")
@useAuth(CloudTokenAuth | CloudCookieAuth)
// Tags will appear in alphabetical order in the API documentation
@tagMetadata(
  "Apps",
  #{
    description: "(Beta) Manage integrations for extending OpenMeter's functionality. ",
  }
)
@tagMetadata(
  "App: Stripe",
  #{ description: "(Beta) Support for Stripe billing." }
)
@tagMetadata(
  "App: Custom Invoicing",
  #{
    description: "(Beta) Interface third party invoicing and payment systems.",
  }
)
@tagMetadata(
  "Billing",
  #{ description: "(Beta) Manage your billing profiles and invoices. " }
)
@tagMetadata(
  "Customers",
  #{
    description: "(Beta) Manage customer subscription lifecycles and plan assignments. ",
  }
)
@tagMetadata("Debug", #{ description: "Debugging and testing endpoints." })
@tagMetadata(
  "Entitlements",
  #{
    description: "(Beta) With Entitlements, you can define and enforce usage limits, implement quota-based pricing, and manage access to features in your application.",
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
  "Meters",
  #{
    description: "Meters specify how to aggregate events for billing and analytics purposes. Meters can be configured with multiple aggregation methods and groupings. Multiple meters can be created for the same event type, enabling flexible metering scenarios.",
  }
)
@tagMetadata(
  "Notifications",
  #{
    description: "(Beta) Notifications provide automated triggers when specific entitlement balances and usage thresholds are reached, ensuring that your customers and sales teams are always informed. Notify customers and internal teams when specific conditions are met, like reaching 75%, 100%, and 150% of their monthly usage allowance. [Read more](https://openmeter.io/docs/guides/notifications/overview).",
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
    description: "(Beta) Configure and manage your product plans, pricing tiers, and subscription offerings. ",
  }
)
@tagMetadata(
  "Subjects",
  #{
    description: "Subjects are entities that consume resources you wish to meter. These can range from users, servers, and services to devices. The design of subjects is intentionally generic, enabling flexible application across various metering scenarios. Typically, a subject acts as a unique identifier within your system for a user or customer. Meters are aggregating events for each subject.",
  }
)
@tagMetadata(
  "Subscriptions",
  #{
    description: "(Beta) With Subscriptions, you can manage your customers' subscriptions. Request access by [contacting us](mailto:contact@openmeter.cloud).",
  }
)
namespace OpenMeterCloud;

@route("/api/v1/events")
@tag("Events")
interface Events extends OpenMeter.Events {}

@route("/api/v2/events")
@tag("Events")
interface EventsV2 extends OpenMeter.EventsV2 {}

@route("/api/v1/meters")
@tag("Meters")
interface Meters extends OpenMeter.Meters {}

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
  @tag("Apps")
  interface Marketplace extends OpenMeter.App.Marketplace {}

  @route("/api/v1/apps")
  @tag("Apps")
  interface Apps extends OpenMeter.App.Apps {}

  @tag("App: Stripe")
  interface AppStripe extends OpenMeter.App.AppStripe {}

  @tag("App: Custom Invoicing")
  interface AppCustomInvoicing extends OpenMeter.App.AppCustomInvoicing {}
}

namespace OpenMeterCloud.Notification {
  @route("/api/v1/notification/channels")
  @tag("Notifications")
  interface Channels extends OpenMeter.Notification.Channels {}

  @route("/api/v1/notification/rules")
  @tag("Notifications")
  interface Rules extends OpenMeter.Notification.Rules {}

  @route("/api/v1/notification/events")
  @tag("Notifications")
  interface Events extends OpenMeter.Notification.Events {}
}

namespace OpenMeterCloud.Entitlements {
  @route("/api/v1/entitlements")
  @tag("Entitlements")
  interface Entitlements extends OpenMeter.Entitlements.Entitlements {}

  @route("/api/v1/features")
  @tag("Entitlements")
  interface Features extends OpenMeter.Entitlements.Features {}

  @route("/api/v1/grants")
  @tag("Entitlements")
  interface Grants extends OpenMeter.Entitlements.Grants {}

  @route("/api/v1/subjects/{subjectIdOrKey}/entitlements")
  @tag("Entitlements")
  interface Subjects extends OpenMeter.Entitlements.Subjects {}

  @route("/api/v1/customers/{customerIdOrKey}")
  @tag("Entitlements")
  @tag("Customers")
  interface Customer extends OpenMeter.Entitlements.Customer {}
}

namespace OpenMeterCloud.Billing {
  @route("/api/v1/customers")
  @tag("Customers")
  interface Customers extends OpenMeter.Customers {}

  @route("/api/v1/billing/customers")
  @tag("Billing")
  interface CustomerOverrides extends OpenMeter.Billing.CustomerOverrides {}

  @route("/api/v1/billing/profiles")
  @tag("Billing")
  interface BillingProfiles extends OpenMeter.Billing.Profiles {}

  @route("/api/v1/billing/invoices")
  @tag("Billing")
  interface InvoicesEndpoints extends OpenMeter.Billing.InvoicesEndpoints {}

  @route("/api/v1/billing/invoices/{invoiceId}")
  @tag("Billing")
  interface InvoiceEndpoints extends OpenMeter.Billing.InvoiceEndpoints {}

  @route("/api/v1/billing/customers/{customerId}/invoices")
  @tag("Billing")
  interface CustomerInvoiceEndpoints
    extends OpenMeter.Billing.CustomerInvoiceEndpoints {}
}

namespace OpenMeterCloud.ProductCatalog {
  @route("/api/v1/plans")
  @tag("Product Catalog")
  interface PlansById extends OpenMeter.ProductCatalog.Plans {}

  @route("/api/v1/addons")
  @tag("Product Catalog")
  interface Addons extends OpenMeter.ProductCatalog.Addons {}

  @route("/api/v1/plans/{planId}/addons")
  @tag("Product Catalog")
  interface PlanAddons extends OpenMeter.ProductCatalog.PlanAddons {}

  @route("/api/v1/subscriptions")
  @tag("Subscriptions")
  interface Subscriptions extends OpenMeter.ProductCatalog.Subscriptions {}

  @route("/api/v1/subscriptions/{subscriptionId}/addons")
  @tag("Subscriptions")
  interface SubscriptionAddons
    extends OpenMeter.ProductCatalog.SubscriptionAddons {}
}

namespace OpenMeterCloud.Subjects {
  @route("/api/v1/subjects")
  @tag("Subjects")
  interface Subjects extends OpenMeter.Subjects {}
}

namespace OpenMeterCloud.Info {
  @route("/api/v1/info/currencies")
  @tag("Lookup Information")
  interface Currencies extends OpenMeter.Info.Currencies {}

  @route("/api/v1/info/progress")
  @tag("Lookup Information")
  interface Progress extends OpenMeter.Info.Progresses {}
}
