package router

import (
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	planhttpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/httpdriver"
)

// List plans
// (GET /api/v1/plans)
func (a *Router) ListPlans(w http.ResponseWriter, r *http.Request, params api.ListPlansParams) {
	if a.config.Plan == nil {
		unimplemented.ListPlans(w, r, params)
		return
	}

	a.planHandler.ListPlans().With(params).ServeHTTP(w, r)
}

// Create a plan
// (POST /api/v1/plans)
func (a *Router) CreatePlan(w http.ResponseWriter, r *http.Request) {
	if a.config.Plan == nil {
		unimplemented.CreatePlan(w, r)
		return
	}

	a.planHandler.CreatePlan().ServeHTTP(w, r)
}

// Delete plan
// (DELETE /api/v1/plans/{planId})
func (a *Router) DeletePlan(w http.ResponseWriter, r *http.Request, planId string) {
	if a.config.Plan == nil {
		unimplemented.DeletePlan(w, r, planId)
		return
	}

	a.planHandler.DeletePlan().With(planId).ServeHTTP(w, r)
}

// Get plan
// (GET /api/v1/plans/{planId})
func (a *Router) GetPlan(w http.ResponseWriter, r *http.Request, planId string, params api.GetPlanParams) {
	if a.config.Plan == nil {
		unimplemented.GetPlan(w, r, planId, params)
		return
	}

	a.planHandler.GetPlan().With(planhttpdriver.GetPlanRequestParams{
		ID:            planId,
		IncludeLatest: lo.FromPtrOr(params.IncludeLatest, false),
	}).ServeHTTP(w, r)
}

// Update a plan
// (PUT /api/v1/plans/{planId})
func (a *Router) UpdatePlan(w http.ResponseWriter, r *http.Request, planId string) {
	if a.config.Plan == nil {
		unimplemented.UpdatePlan(w, r, planId)
		return
	}

	a.planHandler.UpdatePlan().With(planId).ServeHTTP(w, r)
}

// New draft plan
// (POST /api/v1/plans/{planIdOrKey}/next)
func (a *Router) NextPlan(w http.ResponseWriter, r *http.Request, planIdOrKey string) {
	if a.config.Plan == nil {
		unimplemented.NextPlan(w, r, planIdOrKey)
		return
	}

	// TODO: allow key as well
	a.planHandler.NextPlan().With(planIdOrKey).ServeHTTP(w, r)
}

// List phases in plan
// (GET /api/v1/plans/{planId}/phases)
func (a *Router) ListPlanPhases(w http.ResponseWriter, r *http.Request, planId string, params api.ListPlanPhasesParams) {
	if a.config.Plan == nil {
		unimplemented.ListPlanPhases(w, r, planId, params)
		return
	}

	a.planHandler.ListPhases().With(planhttpdriver.ListPhasesParams{
		PlanID:               planId,
		ListPlanPhasesParams: params,
	}).ServeHTTP(w, r)
}

// Create new phase in plan
// (POST /api/v1/plans/{planId}/phases)
func (a *Router) CreatePlanPhase(w http.ResponseWriter, r *http.Request, planId string) {
	if a.config.Plan == nil {
		unimplemented.CreatePlanPhase(w, r, planId)
		return
	}

	a.planHandler.CreatePhase().With(planId).ServeHTTP(w, r)
}

// Delete phase for plan
// (DELETE /api/v1/plans/{planId}/phases/{planPhaseKey})
func (a *Router) DeletePlanPhase(w http.ResponseWriter, r *http.Request, planId string, planPhaseKey string) {
	if a.config.Plan == nil {
		unimplemented.DeletePlanPhase(w, r, planId, planPhaseKey)
		return
	}

	a.planHandler.DeletePhase().With(planhttpdriver.DeletePhaseRequestParams{
		PlanID: planId,
		Key:    planPhaseKey,
	}).ServeHTTP(w, r)
}

// Get phase for plan
// (GET /api/v1/plans/{planId}/phases/{planPhaseKey})
func (a *Router) GetPlanPhase(w http.ResponseWriter, r *http.Request, planId string, planPhaseKey string) {
	if a.config.Plan == nil {
		unimplemented.GetPlanPhase(w, r, planId, planPhaseKey)
		return
	}

	a.planHandler.GetPhase().With(planhttpdriver.PhaseKeyPlanParams{
		PlanID: planId,
		Key:    planPhaseKey,
	}).ServeHTTP(w, r)
}

// Update phase in plan
// (PUT /api/v1/plans/{planId}/phases/{planPhaseKey})
func (a *Router) UpdatePlanPhase(w http.ResponseWriter, r *http.Request, planId string, planPhaseKey string) {
	if a.config.Plan == nil {
		unimplemented.UpdatePlanPhase(w, r, planId, planPhaseKey)
		return
	}

	a.planHandler.UpdatePhase().With(planhttpdriver.PhaseKeyPlanParams{
		PlanID: planId,
		Key:    planPhaseKey,
	}).ServeHTTP(w, r)
}

// Publish plan
// (POST /api/v1/plans/{planId}/publish)
func (a *Router) PublishPlan(w http.ResponseWriter, r *http.Request, planId string) {
	if a.config.Plan == nil {
		unimplemented.PublishPlan(w, r, planId)
		return
	}

	a.planHandler.PublishPlan().With(planId).ServeHTTP(w, r)
}

// Archive plan version
// (POST /api/v1/plans/{planId}/archive)
func (a *Router) ArchivePlan(w http.ResponseWriter, r *http.Request, planId string) {
	if a.config.Plan == nil {
		unimplemented.ArchivePlan(w, r, planId)
		return
	}

	a.planHandler.ArchivePlan().With(planId).ServeHTTP(w, r)
}
