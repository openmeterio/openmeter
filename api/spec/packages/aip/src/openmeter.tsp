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
import "./features/index.tsp";
import "./llmcost/index.tsp";
import "./productcatalog/index.tsp";
import "./organization/index.tsp";

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
@server(
  "http://localhost:{port}/api/v3",
  "Local",
  {
    port?: string = "8888",
  }
)
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
@tagMetadata(Shared.FeaturesTag, #{ description: Shared.FeaturesDescription })
@tagMetadata(Shared.LLMCostTag, #{ description: Shared.LLMCostDescription })
@tagMetadata(
  Shared.ProductCatalogTag,
  #{ description: Shared.ProductCatalogDescription }
)
@tagMetadata(
  Shared.OrganizationTag,
  #{ description: Shared.OrganizationDescription }
)
namespace OpenMeter;

@route("/openmeter/events")
@tag(Shared.EventsTag)
interface EventsEndpoints extends Events.EventsOperations {}

@route("/openmeter/meters")
@tag(Shared.MetersTag)
interface MetersEndpoints extends Meters.MetersOperations {}

@route("/openmeter/meters/{meterId}/query")
@tag(Shared.MetersTag)
interface MetersQueryEndpoints extends Meters.MetersQueryOperations {}

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

@route("/openmeter/customers/{customerId}/credits/grants")
@tag(Shared.CustomersTag)
interface CustomerCreditGrantsEndpoints
  extends Customers.CustomerCreditGrantsOperations {}

@route("/openmeter/customers/{customerId}/credits/balance")
@tag(Shared.CustomersTag)
interface CustomerCreditBalanceEndpoints
  extends Customers.CustomerCreditBalancesOperations {}

@route("/openmeter/customers/{customerId}/credits/adjustments")
@tag(Shared.CustomersTag)
interface CustomerCreditAdjustmentsEndpoints
  extends Customers.CustomerCreditAdjustmentsOperations {}

@route("/openmeter/customers/{customerId}/credits/grants/{creditGrantId}")
@tag(Shared.CustomersTag)
interface CustomerCreditGrantEndpoints {
  // @route("/void")
  // voidGrant is Customers.CustomerCreditGrantVoidOperations.voidGrant;

  @route("/settlement/external")
  updateExternalSettlement is Customers.CustomerCreditGrantExternalSettlementOperations.updateExternalSettlement;
}

@route("/openmeter/customers/{customerId}/credits/transactions")
@tag(Shared.CustomersTag)
interface CustomerCreditTransactionEndpoints
  extends Customers.CustomerCreditTransactionOperations {}

@route("/openmeter/customers/{customerId}/charges")
@tag(Shared.BillingTag)
interface CustomerChargesEndpoints
  extends Customers.CustomerChargesOperations {}

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

@route("/openmeter/currencies/custom")
@tag(Shared.BillingTag)
interface CurrenciesCustomEndpoints
  extends Currencies.CurrenciesCustomOperations {}

@route("/openmeter/currencies/custom/{currencyId}/cost-bases")
@tag(Shared.BillingTag)
interface CurrenciesCustomCostBasesEndpoints
  extends Currencies.CurrenciesCustomCostBasesOperations {}

@route("/openmeter/features")
@tag(Shared.FeaturesTag)
interface FeaturesEndpoints extends Features.FeatureOperations {}

@route("/openmeter/features/{featureId}/cost")
@tag(Shared.FeaturesTag)
interface FeatureCostEndpoints extends Features.FeatureCostOperations {}

@route("/openmeter/llm-cost/prices")
@tag(Shared.LLMCostTag)
interface LLMCostPricesEndpoints extends LLMCost.LLMCostPricesOperations {}

@route("/openmeter/llm-cost/overrides")
@tag(Shared.LLMCostTag)
interface LLMCostOverridesEndpoints
  extends LLMCost.LLMCostOverridesOperations {}

@route("/openmeter/plans")
@tag(Shared.ProductCatalogTag)
interface PlansEndpoints extends ProductCatalog.PlanOperations {}

@route("/openmeter/addons")
@tag(Shared.ProductCatalogTag)
interface AddonsEndpoints extends ProductCatalog.AddonOperations {}

@route("/openmeter/plans/{planId}/addons")
@tag(Shared.ProductCatalogTag)
interface PlanAddonEndpoints extends ProductCatalog.PlanAddonOperations {}

@route("/openmeter/organization/tax-config")
@tag(Shared.OrganizationTag)
interface OrgTaxConfigEndpoints extends Organization.OrgTaxConfigOperations {}
