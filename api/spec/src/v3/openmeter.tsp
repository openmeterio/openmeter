import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "./shared/index.tsp";
import "./events/index.tsp";
import "./meters/index.tsp";
import "./customers/index.tsp";
import "./subscriptions/index.tsp";
import "./apps/index.tsp";

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
@server("https://openmeter.cloud/api/v3", "Cloud")
@tagMetadata(Shared.MetersTag, #{ description: Shared.MetersDescription })
@tagMetadata(Shared.EventsTag, #{ description: Shared.EventsDescription })
@tagMetadata(Shared.CustomersTag, #{ description: Shared.CustomersDescription })
@tagMetadata(
  Shared.SubscriptionsTag,
  #{ description: Shared.SubscriptionsDescription }
)
@tagMetadata(
  Shared.EntitlementsTag,
  #{ description: Shared.EntitlementsDescription }
)
@tagMetadata(Shared.AppsTag, #{ description: Shared.AppsDescription })
namespace OpenMeter;

@route("/openmeter/events")
@tag(Shared.EventsTag)
interface EventsEndpoints extends Events.EventsOperations {}

@route("/openmeter/meters")
@tag(Shared.MetersTag)
interface MetersEndpoints extends Meters.MetersOperations {}

@route("/openmeter/customers")
@tag(Shared.CustomersTag)
interface CustomersEndpoints extends Customers.CustomersOperations {}

@route("/openmeter/customers/{customerId}/entitlement-access")
@tag(Shared.EntitlementsTag)
interface CustomerEntitlementsEndpoints
  extends Entitlements.CustomerEntitlementsOperations {}

@route("/openmeter/subscriptions")
@tag(Shared.SubscriptionsTag)
interface SubscriptionsEndpoints
  extends Subscriptions.SubscriptionsOperations {}

@route("/openmeter/app-catalog")
@tag(Shared.AppsTag)
interface AppCatalogEndpoints extends Apps.AppCatalogOperations {}

@route("/openmeter/apps/custom-invoicing")
@tag(Shared.AppsTag)
interface AppCustomInvoicingEndpoints
  extends Apps.AppCustomInvoicingOperations {}

@route("/openmeter/apps/stripe")
@tag(Shared.AppsTag)
interface AppStripeEndpoints extends Apps.AppStripeOperations {}
