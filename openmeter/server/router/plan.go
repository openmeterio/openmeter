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
	if !a.config.ProductCatalogEnabled {
		unimplemented.ListPlans(w, r, params)
		return
	}

	a.planHandler.ListPlans().With(params).ServeHTTP(w, r)
}

// Create a plan
// (POST /api/v1/plans)
func (a *Router) CreatePlan(w http.ResponseWriter, r *http.Request) {
	if !a.config.ProductCatalogEnabled {
		unimplemented.CreatePlan(w, r)
		return
	}

	a.planHandler.CreatePlan().ServeHTTP(w, r)
}

// Delete plan
// (DELETE /api/v1/plans/{planId})
func (a *Router) DeletePlan(w http.ResponseWriter, r *http.Request, planId string) {
	if !a.config.ProductCatalogEnabled {
		unimplemented.DeletePlan(w, r, planId)
		return
	}

	a.planHandler.DeletePlan().With(planId).ServeHTTP(w, r)
}

// Get plan
// (GET /api/v1/plans/{planId})
func (a *Router) GetPlan(w http.ResponseWriter, r *http.Request, planIdOrKey string, params api.GetPlanParams) {
	if !a.config.ProductCatalogEnabled {
		unimplemented.GetPlan(w, r, planIdOrKey, params)
		return
	}

	a.planHandler.GetPlan().With(planhttpdriver.GetPlanRequestParams{
		IDOrKey:       planIdOrKey,
		IncludeLatest: lo.FromPtrOr(params.IncludeLatest, false),
	}).ServeHTTP(w, r)
}

// Update a plan
// (PUT /api/v1/plans/{planId})
func (a *Router) UpdatePlan(w http.ResponseWriter, r *http.Request, planId string) {
	if !a.config.ProductCatalogEnabled {
		unimplemented.UpdatePlan(w, r, planId)
		return
	}

	a.planHandler.UpdatePlan().With(planId).ServeHTTP(w, r)
}

// Publish plan
// (POST /api/v1/plans/{planId}/publish)
func (a *Router) PublishPlan(w http.ResponseWriter, r *http.Request, planId string) {
	if !a.config.ProductCatalogEnabled {
		unimplemented.PublishPlan(w, r, planId)
		return
	}

	a.planHandler.PublishPlan().With(planId).ServeHTTP(w, r)
}

// Archive plan version
// (POST /api/v1/plans/{planId}/archive)
func (a *Router) ArchivePlan(w http.ResponseWriter, r *http.Request, planId string) {
	if !a.config.ProductCatalogEnabled {
		unimplemented.ArchivePlan(w, r, planId)
		return
	}

	a.planHandler.ArchivePlan().With(planId).ServeHTTP(w, r)
}
