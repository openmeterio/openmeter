import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "./events/index.tsp";
import "./shared/index.tsp";
import "./customers/index.tsp";
import "./entitlements/index.tsp";
import "./subscriptions/index.tsp";

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
@tagMetadata(Shared.SubscriptionsTag, #{ description: Shared.SubscriptionsDescription })
@tagMetadata(Shared.EntitlementsTag, #{ description: Shared.EntitlementsDescription })
namespace OpenMeter;

@route("/openmeter/meters")
@tag(Shared.MetersTag)
@friendlyName("${Shared.OpenMeterTitle}: ${Shared.MetersTag}")
interface MetersEndpoints extends Meters.MetersOperations {}

@route("/openmeter/events")
@tag(Shared.EventsTag)
@friendlyName("${Shared.OpenMeterTitle}: ${Shared.EventsTag}")
interface EventsEndpoints extends Events.EventsOperations {}

@route("/openmeter/customers")
@tag(Shared.CustomersTag)
@friendlyName("${Shared.OpenMeterTitle}: ${Shared.CustomersTag}")
interface CustomersEndpoints extends Customers.CustomersOperations {}

@route("/openmeter/customers/{customerId}/subscriptions")
@tag(Shared.CustomersTag)
@tag(Shared.SubscriptionsTag)
@friendlyName("${Shared.OpenMeterTitle}: ${Shared.CustomersTag}: ${Shared.SubscriptionsTag}")
interface CustomerSubscriptionsEndpoints extends Subscriptions.CustomerSubscriptionsOperations {}

@route("/openmeter/customers/{customerId}/entitlements/{featureKey}")
@tag(Shared.CustomersTag)
@tag(Shared.EntitlementsTag)
@friendlyName("${Shared.OpenMeterTitle}: ${Shared.EntitlementsTag}")
interface CustomerEntitlementsEndpoints
  extends Entitlements.CustomerEntitlementsOperations {}
