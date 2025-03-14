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
	a.planHandler.ListPlans().With(params).ServeHTTP(w, r)
}

// Create a plan
// (POST /api/v1/plans)
func (a *Router) CreatePlan(w http.ResponseWriter, r *http.Request) {
	a.planHandler.CreatePlan().ServeHTTP(w, r)
}

// Delete plan
// (DELETE /api/v1/plans/{planId})
func (a *Router) DeletePlan(w http.ResponseWriter, r *http.Request, planId string) {
	a.planHandler.DeletePlan().With(planId).ServeHTTP(w, r)
}

// Get plan
// (GET /api/v1/plans/{planId})
func (a *Router) GetPlan(w http.ResponseWriter, r *http.Request, planIdOrKey string, params api.GetPlanParams) {
	a.planHandler.GetPlan().With(planhttpdriver.GetPlanRequestParams{
		IDOrKey:       planIdOrKey,
		IncludeLatest: lo.FromPtrOr(params.IncludeLatest, false),
	}).ServeHTTP(w, r)
}

// Update a plan
// (PUT /api/v1/plans/{planId})
func (a *Router) UpdatePlan(w http.ResponseWriter, r *http.Request, planId string) {
	a.planHandler.UpdatePlan().With(planId).ServeHTTP(w, r)
}

// New draft plan
// (POST /api/v1/plans/{planIdOrKey}/next)
func (a *Router) NextPlan(w http.ResponseWriter, r *http.Request, planIdOrKey string) {
	// TODO: allow key as well
	a.planHandler.NextPlan().With(planIdOrKey).ServeHTTP(w, r)
}

// Publish plan
// (POST /api/v1/plans/{planId}/publish)
func (a *Router) PublishPlan(w http.ResponseWriter, r *http.Request, planId string) {
	a.planHandler.PublishPlan().With(planId).ServeHTTP(w, r)
}

// Archive plan version
// (POST /api/v1/plans/{planId}/archive)
func (a *Router) ArchivePlan(w http.ResponseWriter, r *http.Request, planId string) {
	a.planHandler.ArchivePlan().With(planId).ServeHTTP(w, r)
}
