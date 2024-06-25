package router

// We explicitly define no-op implementations for future APIs instead of just using the codegen version.

import (
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
)

var unimplemented api.ServerInterface = api.Unimplemented{}

// Delete entitlement
// (DELETE /api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId})
func (a *Router) DeleteEntitlement(w http.ResponseWriter, r *http.Request, subjectIdOrKey api.SubjectIdOrKey, entitlementId api.EntitlementId) {
	commonhttp.NewHTTPError(http.StatusNotImplemented, fmt.Errorf("not implemented")).EncodeError(r.Context(), w)
}

// Get entitlement
// (GET /api/v1/subjects/{subjectIdOrKey}/entitlements/{entitlementId})
func (a *Router) GetEntitlement(w http.ResponseWriter, r *http.Request, subjectIdOrKey api.SubjectIdOrKey, entitlementId api.EntitlementId) {
	commonhttp.NewHTTPError(http.StatusNotImplemented, fmt.Errorf("not implemented")).EncodeError(r.Context(), w)
}
