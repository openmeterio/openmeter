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
})
@server("https://127.0.0.1:8888", "Local")
@server("https://openmeter.cloud", "Cloud")
@useAuth(CloudTokenAuth | CloudCookieAuth)
@tagMetadata(
  "Meters",
  #{
    description: "Meters are used to aggregate usage events. They are designed to offer multiple aggregation methods and window sizes, making it highly adaptable to your specific needs.",
  }
)
@tagMetadata(
  "Events",
  #{
    description: "A usage event can be anything you need to track accurately over time for billing or analytics purposes.",
  }
)
@tagMetadata(
  "Subjects",
  #{
    description: "Subjects are entities that consume resources you wish to meter. These can range from users, servers, and services to devices. The design of subjects is intentionally generic, enabling flexible application across various metering scenarios. Typically, a subject acts as a unique identifier within your system for a user or customer. [Read more](https://openmeter.io/docs/cloud/subjects).",
  }
)
@tagMetadata(
  "Entitlements (Beta)",
  #{
    description: "Entitlements make it possible to implement complex pricing scenarios such as monthly quotas, prepaid billing, and per-customer pricing. Don't just list usage limits on your marketing page—enforce them within your application to control the cost of expensive resources like LLMs. [Read more](https://openmeter.io/docs/guides/entitlements/overview).",
  }
)
@tagMetadata(
  "Notification (Beta)",
  #{
    description: "Notifications provide automated triggers when specific entitlement balances and usage thresholds are reached, ensuring that your customers and sales teams are always informed. For example, you can email your customers when they reach 75%, 100%, and 150% of their monthly token allowance and notify your sales team when an overage happens to drive expansions. [Read more](https://openmeter.io/docs/guides/notifications/overview).",
  }
)
@tagMetadata(
  "Portal",
  #{
    description: "With the Consumer Portal, you can build in-app user-facing dashboards where your customers can follow their consumption in real-time. [Read more](https://openmeter.io/docs/guides/customer-dashboard).",
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
  interface Tokens extends OpenMeter.Portal.Tokens {
    /**
     * Invalidates consumer portal tokens by ID or subject.
     */
    @route("/invalidate")
    @post
    @operationId("invalidatePortalTokens")
    @summary("Invalidate portal tokens")
    invalidate(
      /**
       * Invalidate a portal token by ID.
       */
      id?: string,

      /**
       * Invalidate all portal tokens for a subject.
       */
      subject?: string,
    ): void | OpenMeter.CommonErrors;
  }

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

  @tag("Apps")
  interface AppStripe extends OpenMeter.App.AppStripe {}
}

namespace OpenMeterCloud.Notification {
  @route("/api/v1/notification/channels")
  @tag("Notification (Beta)")
  interface Channels extends OpenMeter.Notification.Channels {}

  @route("/api/v1/notification/rules")
  @tag("Notification (Beta)")
  interface Rules extends OpenMeter.Notification.Rules {}

  @route("/api/v1/notification/events")
  @tag("Notification (Beta)")
  interface Events extends OpenMeter.Notification.Events {}

  @route("/api/v1/notification/webhook/svix")
  @tag("Notification (Beta)")
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
  @tag("Customers")
  interface Customers extends OpenMeter.Customers {}

  @route("/api/v1/billing/customer")
  @tag("Billing (Experimental)")
  interface CustomerOverrides extends OpenMeter.Billing.CustomerOverrides {}

  @route("/api/v1/billing/profile")
  @tag("Billing (Experimental)")
  interface BillingProfiles extends OpenMeter.Billing.Profiles {}

  @route("/api/v1/billing/invoices")
  @tag("Billing (Experimental)")
  interface Invoices extends OpenMeter.Billing.InvoiceEndpoints {}

  @route("/api/v1/billing/invoices/{customerId}")
  @tag("Billing (Experimental)")
  interface CustomerInvoicesEndpoints
    extends OpenMeter.Billing.CustomerInvoicesEndpoints {}

  @route("/api/v1/billing/invoices/{customerId}/invoices/{invoiceId}")
  @tag("Billing (Experimental)")
  interface CustomerInvoiceEndpoints
    extends OpenMeter.Billing.CustomerInvoiceEndpoints {}
}

namespace OpenMeterCloud.ProductCatalog {
  @route("/api/v1/plans")
  @tag("Plan (Experimental)")
  interface PlansById extends OpenMeter.ProductCatalog.Plans {}

  @route("/api/v1/plans/{planId}/phases")
  @tag("Plan (Experimental)")
  interface PlanByIdPhases extends OpenMeter.ProductCatalog.PlanPhases {}

  @route("/api/v1/subscriptions")
  @tag("Subscriptions")
  interface Subscriptions extends OpenMeter.ProductCatalog.Subscriptions {}
}

/**
 * A meter is a configuration that defines how to match and aggregate events.
 */
@extension("x-go-type", "models.Meter")
@extension(
  "x-go-type-import",
  {
    path: "github.com/openmeterio/openmeter/pkg/models",
  }
)
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
