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
  "Meters",
  #{
    description: "Meters specify how to aggregate events for billing and analytics purposes. Meters can be configured with multiple aggregation methods and groupings. Multiple meters can be created for the same event type, enabling flexible metering scenarios.",
  }
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
namespace OpenMeterCloud;

@route("/api/v1/events")
@tag("Events")
@friendlyName("Events")
interface EventsEndpoints extends OpenMeter.EventsEndpoints {}

@route("/api/v2/events")
@tag("Events")
@friendlyName("EventsV2")
interface EventsV2Endpoints extends OpenMeter.EventsV2Endpoints {}

@route("/api/v1/meters")
@tag("Meters")
@friendlyName("Meters")
interface MetersEndpoints extends OpenMeter.MetersEndpoints {}

namespace OpenMeterCloud.Portal {
  @route("/api/v1/portal/tokens")
  @tag("Portal")
  @friendlyName("PortalTokens")
  interface TokensEndpoints extends OpenMeter.Portal.TokensEndpoints {}

  @route("/api/v1/portal/meters")
  @tag("Portal")
  @useAuth(CloudPortalTokenAuth)
  @friendlyName("PortalMeters")
  interface MetersEndpoints extends OpenMeter.Portal.MetersEndpoints {}
}

@route("/api/v1/debug")
@tag("Debug")
@friendlyName("Debug")
interface DebugEndpoints extends OpenMeter.DebugEndpoints {}

namespace OpenMeterCloud {
  @route("/api/v1/marketplace")
  @tag("Apps")
  @friendlyName("Marketplace")
  interface MarketplaceEndpoints extends OpenMeter.MarketplaceEndpoints {}

  @route("/api/v1/apps")
  @tag("Apps")
  @friendlyName("Apps")
  interface AppsEndpoints extends OpenMeter.AppsEndpoints {}

  @tag("App: Stripe")
  @friendlyName("AppStripe")
  interface AppStripeEndpoints extends OpenMeter.AppStripeEndpoints {}

  @tag("App: Custom Invoicing")
  @friendlyName("AppCustomInvoicing")
  interface AppCustomInvoicingEndpoints
    extends OpenMeter.AppCustomInvoicingEndpoints {}
}

namespace OpenMeterCloud {
  @route("/api/v1/notification/channels")
  @tag("Notifications")
  @friendlyName("NotificationChannels")
  interface ChannelsEndpoints extends OpenMeter.NotificationChannelsEndpoints {}

  @route("/api/v1/notification/rules")
  @tag("Notifications")
  @friendlyName("NotificationRules")
  interface RulesEndpoints extends OpenMeter.NotificationRulesEndpoints {}

  @route("/api/v1/notification/events")
  @tag("Notifications")
  @friendlyName("NotificationEvents")
  interface EventsEndpoints extends OpenMeter.NotificationEventsEndpoints {}
}

namespace OpenMeterCloud {
  @route("/api/v1/entitlements")
  @tag("Entitlements")
  @friendlyName("Entitlements")
  interface EntitlementsEndpoints extends OpenMeter.EntitlementsEndpoints {}

  @route("/api/v1/grants")
  @tag("Entitlements")
  @friendlyName("Grants")
  interface GrantsEndpoints extends OpenMeter.GrantsEndpoints {}

  @route("/api/v1/subjects/{subjectIdOrKey}/entitlements")
  @tag("Entitlements")
  @friendlyName("Subjects")
  interface SubjectEntitlementsEndpoints
    extends OpenMeter.SubjectEntitlementsEndpoints {}

  @route("/api/v1/customers/{customerIdOrKey}")
  @tag("Entitlements")
  @tag("Customers")
  @friendlyName("Customer")
  interface CustomerEndpoints extends OpenMeter.CustomerEndpoints {}

  @route("/api/v1/customers/{customerIdOrKey}/apps")
  @tag("Customers")
  @tag("Apps")
  @friendlyName("CustomerApps")
  interface CustomerAppsEndpoints extends OpenMeter.CustomerAppsEndpoints {}

  @route("/api/v1/customers/{customerIdOrKey}/stripe")
  @tag("Customers")
  @friendlyName("CustomerStripe")
  interface CustomerStripeEndpoints extends OpenMeter.CustomerStripeEndpoints {}

  @route("/api/v1/customers/{customerIdOrKey}/entitlements/{featureKey}")
  @tag("Entitlements")
  @tag("Customers")
  @friendlyName("CustomerEntitlement")
  interface CustomerEntitlementEndpoints
    extends OpenMeter.CustomerEntitlementEndpoints {}

  namespace V2 {
    @route("/api/v2/customers/{customerIdOrKey}/entitlements")
    @tag("Entitlements")
    @tag("Customers")
    @friendlyName("CustomerEntitlementsV2")
    interface CustomerEntitlementsV2Endpoints
      extends OpenMeter.CustomerEntitlementsV2Endpoints {}

    @route("/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}")
    @tag("Entitlements")
    @tag("Customers")
    @friendlyName("CustomerEntitlementV2")
    interface CustomerEntitlementEndpoints
      extends OpenMeter.CustomerEntitlementV2Endpoints {}

    @route("/api/v2/grants")
    @tag("Entitlements")
    @friendlyName("GrantsV2")
    interface GrantsV2Endpoints extends OpenMeter.GrantsV2Endpoints {}

    @route("/api/v2/entitlements")
    @tag("Entitlements")
    @friendlyName("EntitlementsV2")
    interface EntitlementsV2Endpoints
      extends OpenMeter.EntitlementsV2Endpoints {}
  }
}

namespace OpenMeterCloud {
  @route("/api/v1/customers")
  @tag("Customers")
  @friendlyName("Customers")
  interface CustomersEndpoints extends OpenMeter.CustomersEndpoints {}

  @route("/api/v1/billing/customers")
  @tag("Billing")
  @friendlyName("CustomerOverrides")
  interface CustomerOverridesEndpoints
    extends OpenMeter.CustomerOverridesEndpoints {}

  @route("/api/v1/billing/profiles")
  @tag("Billing")
  @friendlyName("BillingProfiles")
  interface BillingProfilesEndpoints
    extends OpenMeter.BillingProfilesEndpoints {}

  @route("/api/v1/billing/invoices")
  @tag("Billing")
  @friendlyName("Invoices")
  interface InvoicesEndpoints extends OpenMeter.InvoicesEndpoints {}

  @route("/api/v1/billing/invoices/{invoiceId}")
  @tag("Billing")
  @friendlyName("Invoice")
  interface InvoiceEndpoints extends OpenMeter.InvoiceEndpoints {}

  @route("/api/v1/billing/customers/{customerId}/invoices")
  @tag("Billing")
  @friendlyName("CustomerInvoice")
  interface CustomerInvoiceEndpoints
    extends OpenMeter.CustomerInvoiceEndpoints {}
}

namespace OpenMeterCloud {
  @route("/api/v1/features")
  @tag("Product Catalog")
  @friendlyName("Features")
  interface FeaturesEndpoints extends OpenMeter.FeaturesEndpoints {}

  @route("/api/v1/plans")
  @tag("Product Catalog")
  @friendlyName("Plans")
  interface PlansEndpoints extends OpenMeter.PlansEndpoints {}

  @route("/api/v1/addons")
  @tag("Product Catalog")
  @friendlyName("Addons")
  interface AddonsEndpoints extends OpenMeter.AddonsEndpoints {}

  @route("/api/v1/plans/{planId}/addons")
  @tag("Product Catalog")
  @friendlyName("PlanAddons")
  interface PlanAddonsEndpoints extends OpenMeter.PlanAddonsEndpoints {}

  @route("/api/v1/subscriptions")
  @tag("Subscriptions")
  @friendlyName("Subscriptions")
  interface SubscriptionsEndpoints extends OpenMeter.SubscriptionsEndpoints {}

  @route("/api/v1/subscriptions/{subscriptionId}/addons")
  @tag("Subscriptions")
  @friendlyName("SubscriptionAddons")
  interface SubscriptionAddonsEndpoints
    extends OpenMeter.SubscriptionAddonsEndpoints {}
}

namespace OpenMeterCloud {
  @route("/api/v1/subjects")
  @tag("Subjects")
  @friendlyName("Subjects")
  interface SubjectsEndpoints extends OpenMeter.SubjectsEndpoints {}
}

namespace OpenMeterCloud {
  @route("/api/v1/info/currencies")
  @tag("Lookup Information")
  @friendlyName("Currencies")
  interface CurrenciesEndpoints extends OpenMeter.CurrenciesEndpoints {}

  @route("/api/v1/info/progress")
  @tag("Lookup Information")
  @friendlyName("Progress")
  interface ProgressEndpoints extends OpenMeter.ProgressEndpoints {}
}
