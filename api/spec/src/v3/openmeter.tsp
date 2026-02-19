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
import "./billing/index.tsp";
import "./currencies/index.tsp";

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
@tagMetadata(Shared.BillingTag, #{ description: Shared.BillingDescription })
@tagMetadata(Shared.TaxTag, #{ description: Shared.TaxDescription })
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

@route("/openmeter/customers/{customerId}/billing")
@tag(Shared.CustomersTag)
interface CustomerBillingEndpoints
  extends Customers.CustomerBillingOperations {}

@route("/openmeter/customers/{customerId}/entitlement-access")
@tag(Shared.EntitlementsTag)
interface CustomerEntitlementsEndpoints
  extends Entitlements.CustomerEntitlementsOperations {}

@route("/openmeter/subscriptions")
@tag(Shared.SubscriptionsTag)
interface SubscriptionsEndpoints
  extends Subscriptions.SubscriptionsOperations {}

@route("/openmeter/apps")
@tag(Shared.AppsTag)
interface AppsEndpoints extends Apps.AppsOperations {}

@route("/openmeter/profiles")
@tag(Shared.BillingTag)
interface BillingProfilesEndpoints extends Billing.BillingProfilesOperations {}

@route("/openmeter/tax-codes")
@tag(Shared.TaxTag)
interface TaxCodesEndpoints extends Tax.TaxCodesOperations {}

@route("/openmeter/currencies")
@tag(Shared.BillingTag)
interface CurrenciesEndpoints extends Currencies.CurrenciesOperations {}

@route("/openmeter/currencies/{currencyId}/cost-bases")
@tag(Shared.BillingTag)
interface CostBasesEndpoints extends Currencies.CostBasesOperations {}
