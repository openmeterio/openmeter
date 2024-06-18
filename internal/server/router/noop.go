package router

// We explicitly define no-op implementations for future APIs instead of just using the codegen version.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
)

var unimplemented api.ServerInterface = api.Unimplemented{}

// List entitlements
// (GET /api/v1/entitlements)
func (a *Router) ListEntitlements(w http.ResponseWriter, r *http.Request, params api.ListEntitlementsParams) {
	commonhttp.NewHTTPError(http.StatusNotImplemented, fmt.Errorf("not implemented")).EncodeError(context.TODO(), w)
}

// Delete entitlement
// (DELETE /api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId})
func (a *Router) DeleteEntitlement(w http.ResponseWriter, r *http.Request, subjectIdOrKey api.SubjectIdOrKey, entitlementId api.EntitlementId) {
	commonhttp.NewHTTPError(http.StatusNotImplemented, fmt.Errorf("not implemented")).EncodeError(context.TODO(), w)
}

// Get entitlement
// (GET /api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId})
func (a *Router) GetEntitlement(w http.ResponseWriter, r *http.Request, subjectIdOrKey api.SubjectIdOrKey, entitlementId api.EntitlementId) {
	commonhttp.NewHTTPError(http.StatusNotImplemented, fmt.Errorf("not implemented")).EncodeError(context.TODO(), w)
}

// List grants for an entitlement
// (GET /api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId}/grants)
func (a *Router) ListEntitlementGrants(w http.ResponseWriter, r *http.Request, subjectIdOrKey api.SubjectIdOrKey, entitlementId api.EntitlementId, params api.ListEntitlementGrantsParams) {
	commonhttp.NewHTTPError(http.StatusNotImplemented, fmt.Errorf("not implemented")).EncodeError(context.TODO(), w)
}

// Get the balance history of a specific entitlement.
// (GET /api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId}/history)
func (a *Router) GetEntitlementHistory(w http.ResponseWriter, r *http.Request, subjectIdOrKey api.SubjectIdOrKey, entitlementId api.EntitlementId, params api.GetEntitlementHistoryParams) {
	commonhttp.NewHTTPError(http.StatusNotImplemented, fmt.Errorf("not implemented")).EncodeError(context.TODO(), w)
}

// Reset entitlement
// (POST /api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId}/reset)
func (a *Router) ResetEntitlementUsage(w http.ResponseWriter, r *http.Request, subjectIdOrKey api.SubjectIdOrKey, entitlementId api.EntitlementId) {
	commonhttp.NewHTTPError(http.StatusNotImplemented, fmt.Errorf("not implemented")).EncodeError(context.TODO(), w)
}

// List grants
// (GET /api/v1/grants)
func (a *Router) ListGrants(w http.ResponseWriter, r *http.Request, params api.ListGrantsParams) {
	commonhttp.NewHTTPError(http.StatusNotImplemented, fmt.Errorf("not implemented")).EncodeError(context.TODO(), w)
}

// Delete a grant
// (DELETE /api/v1/grants/{grantId})
func (a *Router) VoidGrant(w http.ResponseWriter, r *http.Request, grantId api.GrantId) {
	commonhttp.NewHTTPError(http.StatusNotImplemented, fmt.Errorf("not implemented")).EncodeError(context.TODO(), w)
}

// List meters
// (GET /api/v1/meters)
