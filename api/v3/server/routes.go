package server

import (
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	currencieshandler "github.com/openmeterio/openmeter/api/v3/handlers/currencies"
	customerscreditshandler "github.com/openmeterio/openmeter/api/v3/handlers/customers/credits"
)

// Meters

func (s *Server) CreateMeter(w http.ResponseWriter, r *http.Request) {
	s.metersHandler.CreateMeter().ServeHTTP(w, r)
}

func (s *Server) GetMeter(w http.ResponseWriter, r *http.Request, meterId api.ULID) {
	s.metersHandler.GetMeter().With(meterId).ServeHTTP(w, r)
}

func (s *Server) ListMeters(w http.ResponseWriter, r *http.Request, params api.ListMetersParams) {
	s.metersHandler.ListMeters().With(params).ServeHTTP(w, r)
}

func (s *Server) DeleteMeter(w http.ResponseWriter, r *http.Request, meterId api.ULID) {
	s.metersHandler.DeleteMeter().With(meterId).ServeHTTP(w, r)
}

func (s *Server) QueryMeter(w http.ResponseWriter, r *http.Request, meterId api.ULID) {
	s.metersHandler.QueryMeter().With(meterId).ServeHTTP(w, r)
}

// Events

func (s *Server) IngestMeteringEvents(w http.ResponseWriter, r *http.Request) {
	s.eventsHandler.IngestEvents().ServeHTTP(w, r)
}

// Customers

func (s *Server) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	s.customersHandler.CreateCustomer().ServeHTTP(w, r)
}

func (s *Server) GetCustomer(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	s.customersHandler.GetCustomer().With(customerId).ServeHTTP(w, r)
}

func (s *Server) ListCustomers(w http.ResponseWriter, r *http.Request, params api.ListCustomersParams) {
	s.customersHandler.ListCustomers().With(params).ServeHTTP(w, r)
}

func (s *Server) UpsertCustomer(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	s.customersHandler.UpsertCustomer().With(customerId).ServeHTTP(w, r)
}

func (s *Server) DeleteCustomer(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	s.customersHandler.DeleteCustomer().With(customerId).ServeHTTP(w, r)
}

// Customers Entitlement Access

func (s *Server) ListCustomerEntitlementAccess(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	s.customersEntitlementHandler.ListCustomerEntitlementAccess().With(customerId).ServeHTTP(w, r)
}

// Subscriptions

func (s *Server) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	s.subscriptionsHandler.CreateSubscription().ServeHTTP(w, r)
}

func (s *Server) ListSubscriptions(w http.ResponseWriter, r *http.Request, params api.ListSubscriptionsParams) {
	s.subscriptionsHandler.ListSubscriptions().With(params).ServeHTTP(w, r)
}

func (s *Server) GetSubscription(w http.ResponseWriter, r *http.Request, subscriptionId api.ULID) {
	s.subscriptionsHandler.GetSubscription().With(subscriptionId).ServeHTTP(w, r)
}

func (s *Server) CancelSubscription(w http.ResponseWriter, r *http.Request, subscriptionId api.ULID) {
	s.subscriptionsHandler.CancelSubscription().With(subscriptionId).ServeHTTP(w, r)
}

func (s *Server) UnscheduleCancelation(w http.ResponseWriter, r *http.Request, subscriptionId api.ULID) {
	s.subscriptionsHandler.UnscheduleCancelation().With(subscriptionId).ServeHTTP(w, r)
}

func (s *Server) ChangeSubscription(w http.ResponseWriter, r *http.Request, subscriptionId api.ULID) {
	s.subscriptionsHandler.ChangeSubscription().With(subscriptionId).ServeHTTP(w, r)
}

// Apps

func (s *Server) ListApps(w http.ResponseWriter, r *http.Request, params api.ListAppsParams) {
	s.appsHandler.ListApps().With(params).ServeHTTP(w, r)
}

func (s *Server) GetApp(w http.ResponseWriter, r *http.Request, appId api.ULID) {
	s.appsHandler.GetApp().With(appId).ServeHTTP(w, r)
}

// Billing Profiles

func (s *Server) ListBillingProfiles(w http.ResponseWriter, r *http.Request, params api.ListBillingProfilesParams) {
	s.billingProfilesHandler.ListBillingProfiles().With(params).ServeHTTP(w, r)
}

func (s *Server) CreateBillingProfile(w http.ResponseWriter, r *http.Request) {
	s.billingProfilesHandler.CreateBillingProfile().ServeHTTP(w, r)
}

func (s *Server) DeleteBillingProfile(w http.ResponseWriter, r *http.Request, id api.ULID) {
	s.billingProfilesHandler.DeleteBillingProfile().With(id).ServeHTTP(w, r)
}

func (s *Server) GetBillingProfile(w http.ResponseWriter, r *http.Request, id api.ULID) {
	s.billingProfilesHandler.GetBillingProfile().With(id).ServeHTTP(w, r)
}

func (s *Server) UpdateBillingProfile(w http.ResponseWriter, r *http.Request, id api.ULID) {
	s.billingProfilesHandler.UpdateBillingProfile().With(id).ServeHTTP(w, r)
}

// Customer Billing

func (s *Server) GetCustomerBilling(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	s.customersBillingHandler.GetCustomerBilling().With(customerId).ServeHTTP(w, r)
}

func (s *Server) UpdateCustomerBilling(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	s.customersBillingHandler.UpdateCustomerBilling().With(customerId).ServeHTTP(w, r)
}

func (s *Server) UpdateCustomerBillingAppData(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	s.customersBillingHandler.UpdateCustomerBillingAppData().With(customerId).ServeHTTP(w, r)
}

func (s *Server) CreateCustomerStripeCheckoutSession(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	s.customersBillingHandler.CreateCustomerStripeCheckoutSession().With(customerId).ServeHTTP(w, r)
}

func (s *Server) CreateCustomerStripePortalSession(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	s.customersBillingHandler.CreateCustomerStripePortalSession().With(customerId).ServeHTTP(w, r)
}

// Tax Codes

func (s *Server) ListTaxCodes(w http.ResponseWriter, r *http.Request, params api.ListTaxCodesParams) {
	s.taxcodesHandler.ListTaxCodes().With(params).ServeHTTP(w, r)
}

func (s *Server) CreateTaxCode(w http.ResponseWriter, r *http.Request) {
	s.taxcodesHandler.CreateTaxCode().ServeHTTP(w, r)
}

func (s *Server) DeleteTaxCode(w http.ResponseWriter, r *http.Request, taxCodeId api.ULID) {
	s.taxcodesHandler.DeleteTaxCode().With(taxCodeId).ServeHTTP(w, r)
}

func (s *Server) GetTaxCode(w http.ResponseWriter, r *http.Request, taxCodeId api.ULID) {
	s.taxcodesHandler.GetTaxCode().With(taxCodeId).ServeHTTP(w, r)
}

func (s *Server) UpsertTaxCode(w http.ResponseWriter, r *http.Request, taxCodeId api.ULID) {
	s.taxcodesHandler.UpdateTaxCode().With(taxCodeId).ServeHTTP(w, r)
}

// Currencies

func (s *Server) ListCurrencies(w http.ResponseWriter, r *http.Request, params api.ListCurrenciesParams) {
	s.currenciesHandler.ListCurrencies().With(params).ServeHTTP(w, r)
}

func (s *Server) CreateCustomCurrency(w http.ResponseWriter, r *http.Request) {
	s.currenciesHandler.CreateCurrency().ServeHTTP(w, r)
}

func (s *Server) CreateCostBasis(w http.ResponseWriter, r *http.Request, currencyId api.ULID) {
	s.currenciesHandler.CreateCostBasis().With(currencyId).ServeHTTP(w, r)
}

func (s *Server) ListCostBases(w http.ResponseWriter, r *http.Request, currencyId api.ULID, params api.ListCostBasesParams) {
	s.currenciesHandler.ListCostBases().With(currencieshandler.ListCostBasesArgs{CurrencyID: currencyId, Params: params}).ServeHTTP(w, r)
}

// Features

func (s *Server) ListFeatures(w http.ResponseWriter, r *http.Request, params api.ListFeaturesParams) {
	s.featuresHandler.ListFeatures().With(params).ServeHTTP(w, r)
}

func (s *Server) CreateFeature(w http.ResponseWriter, r *http.Request) {
	s.featuresHandler.CreateFeature().ServeHTTP(w, r)
}

func (s *Server) GetFeature(w http.ResponseWriter, r *http.Request, featureId api.ULID) {
	s.featuresHandler.GetFeature().With(featureId).ServeHTTP(w, r)
}

func (s *Server) UpdateFeature(w http.ResponseWriter, r *http.Request, featureId api.ULID) {
	s.featuresHandler.UpdateFeature().With(featureId).ServeHTTP(w, r)
}

func (s *Server) DeleteFeature(w http.ResponseWriter, r *http.Request, featureId api.ULID) {
	s.featuresHandler.DeleteFeature().With(featureId).ServeHTTP(w, r)
}

// Feature Cost

func (s *Server) QueryFeatureCost(w http.ResponseWriter, r *http.Request, featureId api.ULID) {
	s.featureCostHandler.QueryFeatureCost().With(featureId).ServeHTTP(w, r)
}

// LLM Cost Prices

func (s *Server) ListLlmCostPrices(w http.ResponseWriter, r *http.Request, params api.ListLlmCostPricesParams) {
	s.llmcostHandler.ListPrices().With(params).ServeHTTP(w, r)
}

func (s *Server) GetLlmCostPrice(w http.ResponseWriter, r *http.Request, priceId api.ULID) {
	s.llmcostHandler.GetPrice().With(priceId).ServeHTTP(w, r)
}

// LLM Cost Overrides

func (s *Server) ListLlmCostOverrides(w http.ResponseWriter, r *http.Request, params api.ListLlmCostOverridesParams) {
	s.llmcostHandler.ListOverrides().With(params).ServeHTTP(w, r)
}

func (s *Server) CreateLlmCostOverride(w http.ResponseWriter, r *http.Request) {
	s.llmcostHandler.CreateOverride().ServeHTTP(w, r)
}

func (s *Server) DeleteLlmCostOverride(w http.ResponseWriter, r *http.Request, priceId api.ULID) {
	s.llmcostHandler.DeleteOverride().With(priceId).ServeHTTP(w, r)
}

// Plans

func (s *Server) ListPlans(w http.ResponseWriter, r *http.Request, params api.ListPlansParams) {
	s.plansHandler.ListPlans().With(params).ServeHTTP(w, r)
}

func (s *Server) CreatePlan(w http.ResponseWriter, r *http.Request) {
	s.plansHandler.CreatePlan().ServeHTTP(w, r)
}

func (s *Server) GetPlan(w http.ResponseWriter, r *http.Request, planId api.ULID) {
	s.plansHandler.GetPlan().With(planId).ServeHTTP(w, r)
}

func (s *Server) UpdatePlan(w http.ResponseWriter, r *http.Request, planId api.ULID) {
	s.plansHandler.UpdatePlan().With(planId).ServeHTTP(w, r)
}

func (s *Server) DeletePlan(w http.ResponseWriter, r *http.Request, planId api.ULID) {
	s.plansHandler.DeletePlan().With(planId).ServeHTTP(w, r)
}

func (s *Server) ArchivePlan(w http.ResponseWriter, r *http.Request, planId api.ULID) {
	s.plansHandler.ArchivePlan().With(planId).ServeHTTP(w, r)
}

func (s *Server) PublishPlan(w http.ResponseWriter, r *http.Request, planId api.ULID) {
	s.plansHandler.PublishPlan().With(planId).ServeHTTP(w, r)
}

// Addons

func (s *Server) ListAddons(w http.ResponseWriter, r *http.Request, params api.ListAddonsParams) {
	unimplemented.ListAddons(w, r, params)
}

func (s *Server) CreateAddon(w http.ResponseWriter, r *http.Request) {
	unimplemented.CreateAddon(w, r)
}

func (s *Server) GetAddon(w http.ResponseWriter, r *http.Request, addonId api.ULID) {
	unimplemented.GetAddon(w, r, addonId)
}

func (s *Server) UpdateAddon(w http.ResponseWriter, r *http.Request, addonId api.ULID) {
	unimplemented.UpdateAddon(w, r, addonId)
}

func (s *Server) DeleteAddon(w http.ResponseWriter, r *http.Request, addonId api.ULID) {
	unimplemented.DeleteAddon(w, r, addonId)
}

func (s *Server) ArchiveAddon(w http.ResponseWriter, r *http.Request, addonId api.ULID) {
	unimplemented.ArchiveAddon(w, r, addonId)
}

func (s *Server) PublishAddon(w http.ResponseWriter, r *http.Request, addonId api.ULID) {
	unimplemented.PublishAddon(w, r, addonId)
}

// Plan Addons

func (s *Server) ListPlanAddons(w http.ResponseWriter, r *http.Request, planId api.ULID, params api.ListPlanAddonsParams) {
	s.plansHandler.ListPlanAddons().With(params).ServeHTTP(w, r)
}

func (s *Server) CreatePlanAddon(w http.ResponseWriter, r *http.Request, planId api.ULID) {
	s.plansHandler.CreatePlanAddon().With(planId).ServeHTTP(w, r)
}

func (s *Server) GetPlanAddon(w http.ResponseWriter, r *http.Request, planId api.ULID, planAddonId api.ULID) {
	s.plansHandler.GetPlanAddon().With(planAddonId).ServeHTTP(w, r)
}

func (s *Server) UpdatePlanAddon(w http.ResponseWriter, r *http.Request, planId api.ULID, planAddonId api.ULID) {
	s.plansHandler.UpdatePlanAddon().With(planAddonId).ServeHTTP(w, r)
}

func (s *Server) DeletePlanAddon(w http.ResponseWriter, r *http.Request, planId api.ULID, planAddonId api.ULID) {
	s.plansHandler.DeletePlanAddon().With(planAddonId).ServeHTTP(w, r)
}

// Credits

var unimplemented = api.Unimplemented{}

func (s *Server) GetCustomerCreditBalance(w http.ResponseWriter, r *http.Request, customerId api.ULID, params api.GetCustomerCreditBalanceParams) {
	if s.customersCreditsHandler == nil {
		unimplemented.GetCustomerCreditBalance(w, r, customerId, params)
		return
	}

	s.customersCreditsHandler.GetCustomerCreditBalance().With(customerscreditshandler.GetCustomerCreditBalanceParams{
		CustomerID: customerId,
		Params:     params,
	}).ServeHTTP(w, r)
}

func (s *Server) ListCreditGrants(w http.ResponseWriter, r *http.Request, customerId api.ULID, params api.ListCreditGrantsParams) {
	if s.customersCreditsHandler == nil || s.CreditGrantService == nil {
		unimplemented.ListCreditGrants(w, r, customerId, params)
		return
	}

	s.customersCreditsHandler.ListCreditGrants().With(customerscreditshandler.ListCreditGrantsParams{
		CustomerID: customerId,
		Params:     params,
	}).ServeHTTP(w, r)
}

func (s *Server) CreateCreditGrant(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	if s.customersCreditsHandler == nil || s.CreditGrantService == nil {
		unimplemented.CreateCreditGrant(w, r, customerId)
		return
	}

	s.customersCreditsHandler.CreateCreditGrant().With(customerscreditshandler.CreateCreditGrantParams{
		CustomerID: customerId,
	}).ServeHTTP(w, r)
}

func (s *Server) GetCreditGrant(w http.ResponseWriter, r *http.Request, customerId api.ULID, creditGrantId api.ULID) {
	if s.customersCreditsHandler == nil || s.CreditGrantService == nil {
		unimplemented.GetCreditGrant(w, r, customerId, creditGrantId)
		return
	}

	s.customersCreditsHandler.GetCreditGrant().With(customerscreditshandler.GetCreditGrantParams{
		CustomerID:    customerId,
		CreditGrantID: creditGrantId,
	}).ServeHTTP(w, r)
}

func (s *Server) CreateCreditAdjustment(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	unimplemented.CreateCreditAdjustment(w, r, customerId)
}

func (s *Server) UpdateCreditGrantExternalSettlement(w http.ResponseWriter, r *http.Request, customerId api.ULID, creditGrantId api.ULID) {
	unimplemented.UpdateCreditGrantExternalSettlement(w, r, customerId, creditGrantId)
}

func (s *Server) ListCreditTransactions(w http.ResponseWriter, r *http.Request, customerId api.ULID, params api.ListCreditTransactionsParams) {
	unimplemented.ListCreditTransactions(w, r, customerId, params)
}

// Charges

func (s *Server) ListCustomerCharges(w http.ResponseWriter, r *http.Request, customerId api.ULID, params api.ListCustomerChargesParams) {
	unimplemented.ListCustomerCharges(w, r, customerId, params)
}
