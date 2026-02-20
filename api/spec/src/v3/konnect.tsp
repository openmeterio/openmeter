import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "./shared/index.tsp";
import "./events/index.tsp";
import "./meters/index.tsp";
import "./customers/index.tsp";
import "./entitlements/index.tsp";
import "./subscriptions/index.tsp";
import "./apps/index.tsp";
import "./billing/index.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

/**
 * Konnect Metering & Billing API.
 */
@service(#{ title: "Konnect ${Shared.MeteringAndBillingTitle}" })
@info(#{
  version: "0.0.1",
  contact: #{ name: "Kong", url: "https://cloud.konghq.com" },
})
@server("https://us.api.konghq.com/v3", "United-States Production region")
@server("https://eu.api.konghq.com/v3", "Europe Production region")
@server("https://au.api.konghq.com/v3", "Australia Production region")
@server("https://me.api.konghq.com/v3", "Middle-East Production region")
@server("https://in.api.konghq.com/v3", "India Production region")
// TODO: Uncomment this when the Singapore production region is available
// @server("https://sg.api.konghq.com/v3", "Singapore Production region")
@server("https://global.api.konghq.com/v3", "Global Production region")
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
@useAuth(systemAccountAccessToken | personalAccessToken | konnectAccessToken)
namespace MeteringAndBilling;

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

@route("/openmeter/currencies")
@tag(Shared.BillingTag)
interface CurrenciesEndpoints extends Currencies.CurrenciesOperations {}

/**
 * The system account access token is meant for automations and integrations that are not directly associated with a human identity.
 */
#suppress "@openmeter/api-spec/casing-aip" "Use existing values"
@friendlyName("systemAccountAccessToken")
@useRef("../../../../common/definitions/security.yaml#/components/securitySchemes/systemAccountAccessToken")
model systemAccountAccessToken {
  ...Http.BearerAuth;
}

/**
 * The personal access token is meant to be used as an alternative to basic-auth when accessing Konnect via APIs.
 */
#suppress "@openmeter/api-spec/casing-aip" "Use existing values"
@friendlyName("personalAccessToken")
@useRef("../../../../common/definitions/security.yaml#/components/securitySchemes/personalAccessToken")
model personalAccessToken {
  ...Http.BearerAuth;
}

/**
 * The Konnect access token is meant to be used by the Konnect dashboard and the decK CLI authenticate with.
 */
#suppress "@openmeter/api-spec/casing-aip" "Use existing values"
@friendlyName("konnectAccessToken")
@useRef("../../../../common/definitions/security.yaml#/components/securitySchemes/konnectAccessToken")
model konnectAccessToken {
  ...Http.BearerAuth;
}
