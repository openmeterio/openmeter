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
