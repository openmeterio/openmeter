package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
	customerdriver "github.com/openmeterio/openmeter/openmeter/customer/httpdriver"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
)

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

// List entitlements
// (GET /api/v1/entitlements)
func (a *Router) ListEntitlements(w http.ResponseWriter, r *http.Request, params api.ListEntitlementsParams) {
	a.entitlementHandler.ListEntitlements().With(params).ServeHTTP(w, r)
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

// Get an entitlement
// (GET /api/v1/entitlements/{entitlementId})
func (a *Router) GetEntitlementById(w http.ResponseWriter, r *http.Request, entitlementId string) {
	a.entitlementHandler.GetEntitlementById().With(entitlementdriver.GetEntitlementByIdHandlerParams{
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

// Customer APIs

// Get customer access
// (GET /api/v1/customers/{customerId}/access)
func (a *Router) GetCustomerAccess(w http.ResponseWriter, r *http.Request, customerIdOrKey string) {
	a.customerHandler.GetCustomerAccess().With(customerdriver.GetCustomerAccessParams{
		CustomerIDOrKey: customerIdOrKey,
	}).ServeHTTP(w, r)
}

// Create customer entitlement
// (POST /api/v1/customers/{customerIdOrKey}/entitlements)
func (a *Router) CreateCustomerEntitlement(w http.ResponseWriter, r *http.Request, customerIdOrKey string) {
	unimplemented.CreateCustomerEntitlement(w, r, customerIdOrKey)
}

// List customer entitlements
// (GET /api/v1/customers/{customerIdOrKey}/entitlements)
func (a *Router) ListCustomerEntitlements(w http.ResponseWriter, r *http.Request, customerIdOrKey string, params api.ListCustomerEntitlementsParams) {
	unimplemented.ListCustomerEntitlements(w, r, customerIdOrKey, params)
}

// Get customer entitlement
// (GET /api/v1/customers/{customerIdOrKey}/entitlements/{featureKey})
func (a *Router) GetCustomerEntitlement(w http.ResponseWriter, r *http.Request, customerIdOrKey string, featureKey string) {
	unimplemented.GetCustomerEntitlement(w, r, customerIdOrKey, featureKey)
}

// Delete customer entitlement
// (DELETE /api/v1/customers/{customerIdOrKey}/entitlements/{featureKey})
func (a *Router) DeleteCustomerEntitlement(w http.ResponseWriter, r *http.Request, customerIdOrKey string, featureKey string) {
	unimplemented.DeleteCustomerEntitlement(w, r, customerIdOrKey, featureKey)
}

// Override customer entitlement
// (PUT /api/v1/customers/{customerIdOrKey}/entitlements/{featureKey}/override)
func (a *Router) OverrideCustomerEntitlement(w http.ResponseWriter, r *http.Request, customerIdOrKey string, featureKey string) {
	unimplemented.OverrideCustomerEntitlement(w, r, customerIdOrKey, featureKey)
}

// List customer entitlement grants
// (GET /api/v1/customers/{customerIdOrKey}/entitlements/{featureKey}/grants)
func (a *Router) ListCustomerEntitlementGrants(w http.ResponseWriter, r *http.Request, customerIdOrKey string, featureKey string, params api.ListCustomerEntitlementGrantsParams) {
	unimplemented.ListCustomerEntitlementGrants(w, r, customerIdOrKey, featureKey, params)
}

// Create customer entitlement grant
// (POST /api/v1/customers/{customerIdOrKey}/entitlements/{featureKey}/grants)
func (a *Router) CreateCustomerEntitlementGrant(w http.ResponseWriter, r *http.Request, customerIdOrKey string, featureKey string) {
	unimplemented.CreateCustomerEntitlementGrant(w, r, customerIdOrKey, featureKey)
}

// Get entitlement value
// (GET /api/v1/customers/{customerId}/entitlements/{featureKey}/value)
func (a *Router) GetCustomerEntitlementValue(w http.ResponseWriter, r *http.Request, customerIdOrKey string, featureKey string, params api.GetCustomerEntitlementValueParams) {
	a.customerHandler.GetCustomerEntitlementValue().With(customerdriver.GetCustomerEntitlementValueParams{
		CustomerIDOrKey: customerIdOrKey,
		FeatureKey:      featureKey,
	}).ServeHTTP(w, r)
}

// Get entitlement history
// (GET /api/v1/customers/{customerId}/entitlements/{featureKey}/history)
func (a *Router) GetCustomerEntitlementHistory(w http.ResponseWriter, r *http.Request, customerIdOrKey string, featureKey string, params api.GetCustomerEntitlementHistoryParams) {
	unimplemented.GetCustomerEntitlementHistory(w, r, customerIdOrKey, featureKey, params)
}

// Reset entitlement usage
// (POST /api/v1/customers/{customerId}/entitlements/{featureKey}/reset)
func (a *Router) ResetCustomerEntitlementUsage(w http.ResponseWriter, r *http.Request, customerIdOrKey string, featureKey string) {
	unimplemented.ResetCustomerEntitlementUsage(w, r, customerIdOrKey, featureKey)
}
