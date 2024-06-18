package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/entitlement/httpdriver"
)

// Create entitlement
// (POST /api/v1/subjects/{subjectIdOrKey}/entitlements)
func (a *Router) CreateEntitlement(w http.ResponseWriter, r *http.Request, subjectIdOrKey api.SubjectIdOrKey) {
	if !a.config.EntitlementsEnabled {
		unimplemented.CreateFeature(w, r)
		return
	}
	a.entitlementHandler.CreateEntitlement().With(subjectIdOrKey).ServeHTTP(w, r)
}

// List entitlements
// (GET /api/v1/subjects/{subjectIdOrKey}/entitlements)
func (a *Router) ListSubjectEntitlements(w http.ResponseWriter, r *http.Request, subjectIdOrKey string, params api.ListSubjectEntitlementsParams) {
	if !a.config.EntitlementsEnabled {
		unimplemented.ListSubjectEntitlements(w, r, subjectIdOrKey, params)
		return
	}
	a.entitlementHandler.GetEntitlementsOfSubjectHandler().With(httpdriver.GetEntitlementsOfSubjectParams{
		SubjectIdOrKey: subjectIdOrKey,
		Params:         params,
	}).ServeHTTP(w, r)
}

// Get the value of a specific entitlement.
// (GET /api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/value)
func (a *Router) GetEntitlementValue(w http.ResponseWriter, r *http.Request, subjectIdOrKey api.SubjectIdOrKey, entitlementIdOrFeatureKey api.EntitlementIdOrFeatureKey, params api.GetEntitlementValueParams) {
	if !a.config.EntitlementsEnabled {
		unimplemented.GetEntitlementValue(w, r, subjectIdOrKey, entitlementIdOrFeatureKey, params)
		return
	}
	a.entitlementHandler.GetEntitlementValue().With(httpdriver.GetEntitlementValueParams{
		SubjectIdOrKey:            subjectIdOrKey,
		EntitlementIdOrFeatureKey: entitlementIdOrFeatureKey,
		Params:                    params,
	}).ServeHTTP(w, r)
}

// Create grant
// (POST /api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId}/grants)
func (a *Router) CreateGrant(w http.ResponseWriter, r *http.Request, subjectIdOrKey api.SubjectIdOrKey, entitlementId api.EntitlementId) {
	if !a.config.EntitlementsEnabled {
		unimplemented.CreateGrant(w, r, subjectIdOrKey, entitlementId)
		return
	}
	a.meteredEntitlementHandler.CreateGrant().With(httpdriver.CreateGrantParams{
		SubjectKey:    subjectIdOrKey,
		EntitlementID: entitlementId,
	}).ServeHTTP(w, r)
}

// List grants for an entitlement
// (GET /api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId}/grants)
func (a *Router) ListEntitlementGrants(w http.ResponseWriter, r *http.Request, subjectIdOrKey api.SubjectIdOrKey, entitlementId api.EntitlementId, params api.ListEntitlementGrantsParams) {
	if !a.config.EntitlementsEnabled {
		unimplemented.ListEntitlementGrants(w, r, subjectIdOrKey, entitlementId, params)
		return
	}
	a.meteredEntitlementHandler.ListEntitlementGrants().With(httpdriver.ListEntitlementGrantsParams{
		SubjectKey:    subjectIdOrKey,
		EntitlementID: entitlementId,
	}).ServeHTTP(w, r)
}

// Reset entitlement
// (POST /api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId}/reset)
func (a *Router) ResetEntitlementUsage(w http.ResponseWriter, r *http.Request, subjectIdOrKey api.SubjectIdOrKey, entitlementId api.EntitlementId) {
	if !a.config.EntitlementsEnabled {
		unimplemented.ResetEntitlementUsage(w, r, subjectIdOrKey, entitlementId)
		return
	}
	a.meteredEntitlementHandler.ResetEntitlementUsage().With(httpdriver.ResetEntitlementUsageParams{
		SubjectKey:    subjectIdOrKey,
		EntitlementID: entitlementId,
	}).ServeHTTP(w, r)
}
