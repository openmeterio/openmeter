package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
)

// List plans
// (GET /api/v1/plans)
func (a *Router) ListPlans(w http.ResponseWriter, r *http.Request, params api.ListPlansParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Create a plan
// (POST /api/v1/plans)
func (a *Router) CreatePlan(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Delete a notification channel
// (DELETE /api/v1/plans/{planId})
func (a *Router) DeletePlan(w http.ResponseWriter, r *http.Request, planId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Get notification channel
// (GET /api/v1/plans/{planId})
func (a *Router) GetPlan(w http.ResponseWriter, r *http.Request, planId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Update a plan
// (PUT /api/v1/plans/{planId})
func (a *Router) UpdatePlan(w http.ResponseWriter, r *http.Request, planId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// List phases in plan
// (GET /api/v1/plans/{planId}/phases)
func (a *Router) ListPlanByIdPhases(w http.ResponseWriter, r *http.Request, planId string, params api.ListPlanByIdPhasesParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Create new phase in plan
// (POST /api/v1/plans/{planId}/phases)
func (a *Router) CreatePlanByIdPhases(w http.ResponseWriter, r *http.Request, planId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Delete phase in plan
// (DELETE /api/v1/plans/{planId}/phases/{planPhaseKey})
func (a *Router) DeletePlanByIdPhases(w http.ResponseWriter, r *http.Request, planId string, planPhaseKey string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Get phase in plan
// (GET /api/v1/plans/{planId}/phases/{planPhaseKey})
func (a *Router) GetPlanByIdPhases(w http.ResponseWriter, r *http.Request, planId string, planPhaseKey string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Update phase in plan
// (PUT /api/v1/plans/{planId}/phases/{planPhaseKey})
func (a *Router) UpdatePlanByIdPhases(w http.ResponseWriter, r *http.Request, planId string, planPhaseKey string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Publish plan
// (POST /api/v1/plans/{planId}/publish)
func (a *Router) PublishPlanById(w http.ResponseWriter, r *http.Request, planId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Unpublish plan
// (POST /api/v1/plans/{planId}/unpublish)
func (a *Router) UnpublishPlanById(w http.ResponseWriter, r *http.Request, planId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// List plans
// (GET /api/v1/plans/{planKey}/versions)
func (a *Router) ListPlansByKeyVersion(w http.ResponseWriter, r *http.Request, planKey string, params api.ListPlansByKeyVersionParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Delete plan
// (DELETE /api/v1/plans/{planKey}/versions/{planVersion})
func (a *Router) DeletePlanByKeyVersion(w http.ResponseWriter, r *http.Request, planKey string, planVersion int) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Get plan
// (GET /api/v1/plans/{planKey}/versions/{planVersion})
func (a *Router) GetPlanByKeyVersion(w http.ResponseWriter, r *http.Request, planKey string, planVersion int) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Update plan
// (PUT /api/v1/plans/{planKey}/versions/{planVersion})
func (a *Router) UpdatePlanByKeyVersion(w http.ResponseWriter, r *http.Request, planKey string, planVersion int) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Delete plan
// (DELETE /api/v1/plans/{planKey}/versions/{planVersion}/phases)
func (a *Router) DeletePlanByKeyVersionPhases(w http.ResponseWriter, r *http.Request, planKey string, planVersion int) {
	w.WriteHeader(http.StatusNotImplemented)
}

// List plans
// (GET /api/v1/plans/{planKey}/versions/{planVersion}/phases)
func (a *Router) ListPlanByKeyVersionPhases(w http.ResponseWriter, r *http.Request, planKey string, planVersion int, params api.ListPlanByKeyVersionPhasesParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Create new phase in plan
// (POST /api/v1/plans/{planKey}/versions/{planVersion}/phases)
func (a *Router) CreatePlanByKeyVersionPhases(w http.ResponseWriter, r *http.Request, planKey string, planVersion int) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Get plan
// (GET /api/v1/plans/{planKey}/versions/{planVersion}/phases/{phaseKey})
func (a *Router) GetPlanByKeyVersionPhases(w http.ResponseWriter, r *http.Request, planKey string, planVersion int, phaseKey string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Update plan
// (PUT /api/v1/plans/{planKey}/versions/{planVersion}/phases/{phaseKey})
func (a *Router) UpdatePlanByKeyVersionPhases(w http.ResponseWriter, r *http.Request, planKey string, planVersion int, phaseKey string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Publish plan
// (POST /api/v1/plans/{planKey}/versions/{planVersion}/publish)
func (a *Router) PublishPlanByKeyVersion(w http.ResponseWriter, r *http.Request, planKey string, planVersion int) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Unpublish plan
// (POST /api/v1/plans/{planKey}/versions/{planVersion}/unpublish)
func (a *Router) UnpublishPlanByKeyVersion(w http.ResponseWriter, r *http.Request, planKey string, planVersion int) {
	w.WriteHeader(http.StatusNotImplemented)
}
