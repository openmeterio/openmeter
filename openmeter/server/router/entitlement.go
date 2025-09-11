package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
	customerdriver "github.com/openmeterio/openmeter/openmeter/customer/httpdriver"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	entitlementdriverv2 "github.com/openmeterio/openmeter/openmeter/entitlement/driver/v2"
)

// ------------------------------------------------------------
// Entitlement APIs (V1)
// ------------------------------------------------------------

// List entitlements
// (GET /api/v1/entitlements)
func (a *Router) ListEntitlements(w http.ResponseWriter, r *http.Request, params api.ListEntitlementsParams) {
	a.entitlementHandler.ListEntitlements().With(params).ServeHTTP(w, r)
}

// Get an entitlement
// (GET /api/v1/entitlements/{entitlementId})
func (a *Router) GetEntitlementById(w http.ResponseWriter, r *http.Request, entitlementId string) {
	a.entitlementHandler.GetEntitlementById().With(entitlementdriver.GetEntitlementByIdHandlerParams{
		EntitlementId: entitlementId,
	}).ServeHTTP(w, r)
}

// ------------------------------------------------------------
// Subject Entitlement APIs (V1)
// ------------------------------------------------------------

// Create entitlement
// (POST /api/v1/subjects/{subjectIdOrKey}/entitlements)
func (a *Router) CreateEntitlement(w http.ResponseWriter, r *http.Request, subjectIdOrKey string) {
	a.entitlementHandler.CreateEntitlement().With(subjectIdOrKey).ServeHTTP(w, r)
}

// List entitlements
// (GET /api/v1/subjects/{subjectIdOrKey}/entitlements)
func (a *Router) ListSubjectEntitlements(w http.ResponseWriter, r *http.Request, subjectIdOrKey string, params api.ListSubjectEntitlementsParams) {
	a.entitlementHandler.GetEntitlementsOfSubjectHandler().With(entitlementdriver.GetEntitlementsOfSubjectHandlerParams{
		SubjectIdOrKey: subjectIdOrKey,
		Params:         params,
	}).ServeHTTP(w, r)
}

// Get the value of a specific entitlement.
// (GET /api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/value)
func (a *Router) GetEntitlementValue(w http.ResponseWriter, r *http.Request, subjectIdOrKey string, entitlementIdOrFeatureKey string, params api.GetEntitlementValueParams) {
	a.entitlementHandler.GetEntitlementValue().With(entitlementdriver.GetEntitlementValueHandlerParams{
		SubjectKey:                subjectIdOrKey,
		EntitlementIdOrFeatureKey: entitlementIdOrFeatureKey,
		Params:                    params,
	}).ServeHTTP(w, r)
}

// Create grant
// (POST /api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId}/grants)
func (a *Router) CreateGrant(w http.ResponseWriter, r *http.Request, subjectIdOrKey string, entitlementIdOrFeatureKey string) {
	a.meteredEntitlementHandler.CreateGrant().With(entitlementdriver.CreateGrantHandlerParams{
		SubjectKey:                subjectIdOrKey,
		EntitlementIdOrFeatureKey: entitlementIdOrFeatureKey,
	}).ServeHTTP(w, r)
}

// List grants for an entitlement
// (GET /api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants)
func (a *Router) ListEntitlementGrants(w http.ResponseWriter, r *http.Request, subjectIdOrKey string, entitlementIdOrFeatureKey string, params api.ListEntitlementGrantsParams) {
	a.meteredEntitlementHandler.ListEntitlementGrants().With(entitlementdriver.ListEntitlementGrantsHandlerParams{
		SubjectKey:                subjectIdOrKey,
		EntitlementIdOrFeatureKey: entitlementIdOrFeatureKey,
	}).ServeHTTP(w, r)
}

// Reset entitlement
// (POST /api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId}/reset)
func (a *Router) ResetEntitlementUsage(w http.ResponseWriter, r *http.Request, subjectIdOrKey string, entitlementId string) {
	a.meteredEntitlementHandler.ResetEntitlementUsage().With(entitlementdriver.ResetEntitlementUsageHandlerParams{
		SubjectKey:    subjectIdOrKey,
		EntitlementID: entitlementId,
	}).ServeHTTP(w, r)
}

// Get the balance history of a specific entitlement.
// (GET /api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId}/history)
func (a *Router) GetEntitlementHistory(w http.ResponseWriter, r *http.Request, subjectIdOrKey string, entitlementId string, params api.GetEntitlementHistoryParams) {
	a.meteredEntitlementHandler.GetEntitlementBalanceHistory().With(entitlementdriver.GetEntitlementBalanceHistoryHandlerParams{
		EntitlementID: entitlementId,
		SubjectKey:    subjectIdOrKey,
		Params:        params,
	}).ServeHTTP(w, r)
}

// Delete entitlement
// (DELETE /api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId})
func (a *Router) DeleteEntitlement(w http.ResponseWriter, r *http.Request, subjectIdOrKey string, entitlementId string) {
	a.entitlementHandler.DeleteEntitlement().With(entitlementdriver.DeleteEntitlementHandlerParams{
		EntitlementId: entitlementId,
	}).ServeHTTP(w, r)
}

// Get entitlement
// (GET /api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId})
func (a *Router) GetEntitlement(w http.ResponseWriter, r *http.Request, subjectIdOrKey string, entitlementId string) {
	a.entitlementHandler.GetEntitlement().With(entitlementdriver.GetEntitlementHandlerParams{
		EntitlementId: entitlementId,
	}).ServeHTTP(w, r)
}

// Override an entitlement
// (PUT /api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/override)
func (a *Router) OverrideEntitlement(w http.ResponseWriter, r *http.Request, subjectIdOrKey string, entitlementIdOrFeatureKey string) {
	a.entitlementHandler.OverrideEntitlement().With(entitlementdriver.OverrideEntitlementHandlerParams{
		SubjectIdOrKey:            subjectIdOrKey,
		EntitlementIdOrFeatureKey: entitlementIdOrFeatureKey,
	}).ServeHTTP(w, r)
}

// ------------------------------------------------------------
// Customer Entitlement APIs (V1)
// ------------------------------------------------------------

// Get entitlement value
// (GET /api/v1/customers/{customerId}/entitlements/{featureKey}/value)
func (a *Router) GetCustomerEntitlementValue(w http.ResponseWriter, r *http.Request, customerIdOrKey string, featureKey string, params api.GetCustomerEntitlementValueParams) {
	a.customerHandler.GetCustomerEntitlementValue().With(customerdriver.GetCustomerEntitlementValueParams{
		CustomerIDOrKey: customerIdOrKey,
		FeatureKey:      featureKey,
	}).ServeHTTP(w, r)
}

// Get customer access
// (GET /api/v2/customers/{customerId}/access)
func (a *Router) GetCustomerAccess(w http.ResponseWriter, r *http.Request, customerIdOrKey string) {
	a.customerHandler.GetCustomerAccess().With(customerdriver.GetCustomerAccessParams{
		CustomerIDOrKey: customerIdOrKey,
	}).ServeHTTP(w, r)
}

// ------------------------------------------------------------
// Entitlement APIs (V2)
// ------------------------------------------------------------

func (a *Router) ListEntitlementsV2(w http.ResponseWriter, r *http.Request, params api.ListEntitlementsV2Params) {
	a.entitlementV2Handler.ListEntitlements().With(params).ServeHTTP(w, r)
}

func (a *Router) GetEntitlementByIdV2(w http.ResponseWriter, r *http.Request, entitlementId string) {
	a.entitlementV2Handler.GetEntitlement().With(entitlementdriverv2.GetEntitlementHandlerParams{
		EntitlementId: entitlementId,
	}).ServeHTTP(w, r)
}

// ------------------------------------------------------------
// Customer Entitlement APIs (V2)
// ------------------------------------------------------------

// Create customer entitlement
// (POST /api/v2/customers/{customerIdOrKey}/entitlements)
func (a *Router) CreateCustomerEntitlementV2(w http.ResponseWriter, r *http.Request, customerIdOrKey api.ULIDOrExternalKey) {
	a.entitlementV2Handler.CreateCustomerEntitlement().With(customerIdOrKey).ServeHTTP(w, r)
}

// List customer entitlements
// (GET /api/v2/customers/{customerIdOrKey}/entitlements)
func (a *Router) ListCustomerEntitlementsV2(w http.ResponseWriter, r *http.Request, customerIdOrKey api.ULIDOrExternalKey, params api.ListCustomerEntitlementsV2Params) {
	a.entitlementV2Handler.ListCustomerEntitlements().With(entitlementdriverv2.ListCustomerEntitlementsHandlerParams{
		CustomerIdOrKey: customerIdOrKey,
		Params:          params,
	}).ServeHTTP(w, r)
}

// Get customer entitlement
// (GET /api/v2/customers/{customerIdOrKey}/entitlements/{featureKey})
func (a *Router) GetCustomerEntitlementV2(w http.ResponseWriter, r *http.Request, customerIdOrKey string, featureKey string) {
	a.entitlementV2Handler.GetCustomerEntitlement().With(entitlementdriverv2.GetCustomerEntitlementHandlerParams{
		CustomerIDOrKey:           customerIdOrKey,
		EntitlementIdOrFeatureKey: featureKey,
	}).ServeHTTP(w, r)
}

// Delete customer entitlement
// (DELETE /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey})
func (a *Router) DeleteCustomerEntitlementV2(w http.ResponseWriter, r *http.Request, customerIdOrKey string, entitlementIdOrFeatureKey string) {
	a.entitlementV2Handler.DeleteCustomerEntitlement().With(entitlementdriverv2.DeleteCustomerEntitlementHandlerParams{
		CustomerIDOrKey:           customerIdOrKey,
		EntitlementIdOrFeatureKey: entitlementIdOrFeatureKey,
	}).ServeHTTP(w, r)
}

// Override customer entitlement
// (PUT /api/v2/customers/{customerIdOrKey}/entitlements/{featureKey}/override)
func (a *Router) OverrideCustomerEntitlementV2(w http.ResponseWriter, r *http.Request, customerIdOrKey string, featureKey string) {
	a.entitlementV2Handler.OverrideCustomerEntitlement().With(entitlementdriverv2.OverrideCustomerEntitlementHandlerParams{
		CustomerIDOrKey:           customerIdOrKey,
		EntitlementIdOrFeatureKey: featureKey,
	}).ServeHTTP(w, r)
}

// List customer entitlement grants
// (GET /api/v2/customers/{customerIdOrKey}/entitlements/{featureKey}/grants)
func (a *Router) ListCustomerEntitlementGrantsV2(w http.ResponseWriter, r *http.Request, customerIdOrKey string, featureKey string, params api.ListCustomerEntitlementGrantsV2Params) {
	a.entitlementV2Handler.ListCustomerEntitlementGrants().With(entitlementdriverv2.ListCustomerEntitlementGrantsHandlerParams{
		CustomerIDOrKey:           customerIdOrKey,
		EntitlementIdOrFeatureKey: featureKey,
		Params:                    params,
	}).ServeHTTP(w, r)
}

// Create customer entitlement grant
// (POST /api/v2/customers/{customerIdOrKey}/entitlements/{featureKey}/grants)
func (a *Router) CreateCustomerEntitlementGrantV2(w http.ResponseWriter, r *http.Request, customerIdOrKey string, featureKey string) {
	a.entitlementV2Handler.CreateCustomerEntitlementGrant().With(entitlementdriverv2.CreateCustomerEntitlementGrantHandlerParams{
		CustomerIDOrKey:           customerIdOrKey,
		EntitlementIdOrFeatureKey: featureKey,
	}).ServeHTTP(w, r)
}

// Get entitlement value
// (GET /api/v2/customers/{customerId}/entitlements/{featureKey}/value)
func (a *Router) GetCustomerEntitlementValueV2(w http.ResponseWriter, r *http.Request, customerIdOrKey string, featureKey string, params api.GetCustomerEntitlementValueV2Params) {
	a.customerHandler.GetCustomerEntitlementValue().With(customerdriver.GetCustomerEntitlementValueParams{
		CustomerIDOrKey: customerIdOrKey,
		FeatureKey:      featureKey,
	}).ServeHTTP(w, r)
}

// Get entitlement history
// (GET /api/v2/customers/{customerId}/entitlements/{featureKey}/history)
func (a *Router) GetCustomerEntitlementHistoryV2(w http.ResponseWriter, r *http.Request, customerIdOrKey string, featureKey string, params api.GetCustomerEntitlementHistoryV2Params) {
	a.entitlementV2Handler.GetCustomerEntitlementHistory().With(entitlementdriverv2.GetCustomerEntitlementHistoryHandlerParams{
		CustomerIDOrKey:           customerIdOrKey,
		EntitlementIdOrFeatureKey: featureKey,
		Params:                    params,
	}).ServeHTTP(w, r)
}

// Reset entitlement usage
// (POST /api/v2/customers/{customerId}/entitlements/{featureKey}/reset)
func (a *Router) ResetCustomerEntitlementUsageV2(w http.ResponseWriter, r *http.Request, customerIdOrKey string, featureKey string) {
	a.entitlementV2Handler.ResetCustomerEntitlementUsage().With(entitlementdriverv2.ResetCustomerEntitlementUsageHandlerParams{
		CustomerIDOrKey:           customerIdOrKey,
		EntitlementIdOrFeatureKey: featureKey,
	}).ServeHTTP(w, r)
}
