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
