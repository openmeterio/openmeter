package router

import (
	"net/http"

	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	planhttpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/httpdriver"
	"github.com/openmeterio/openmeter/pkg/featuregate"
)

// List plans
// (GET /api/v1/plans)
func (a *Router) ListPlans(w http.ResponseWriter, r *http.Request, params apiv1models.ListPlansParams) {
	a.planHandler.ListPlans().With(params).ServeHTTP(w, r)
}

// Create a plan
// (POST /api/v1/plans)
func (a *Router) CreatePlan(w http.ResponseWriter, r *http.Request) {
	a.planHandler.CreatePlan().
		Chain(featuregate.NewMiddleware[planhttpdriver.CreatePlanHandlerRequest, planhttpdriver.CreatePlanResponse](
			a.config.NamespaceDecoder.GetNamespace,
			a.config.FeatureGate,
		)).ServeHTTP(w, r)
}

// Delete plan
// (DELETE /api/v1/plans/{planId})
func (a *Router) DeletePlan(w http.ResponseWriter, r *http.Request, planId string) {
	a.planHandler.DeletePlan().With(planId).ServeHTTP(w, r)
}

// Get plan
// (GET /api/v1/plans/{planId})
func (a *Router) GetPlan(w http.ResponseWriter, r *http.Request, planIdOrKey string, params apiv1models.GetPlanParams) {
	a.planHandler.GetPlan().With(planhttpdriver.GetPlanRequestParams{
		IDOrKey:       planIdOrKey,
		IncludeLatest: lo.FromPtrOr(params.IncludeLatest, false),
	}).ServeHTTP(w, r)
}

// Update a plan
// (PUT /api/v1/plans/{planId})
func (a *Router) UpdatePlan(w http.ResponseWriter, r *http.Request, planId string) {
	a.planHandler.UpdatePlan().
		Chain(featuregate.NewMiddleware[planhttpdriver.UpdatePlanHandlerRequest, planhttpdriver.UpdatePlanResponse](
			a.config.NamespaceDecoder.GetNamespace,
			a.config.FeatureGate,
		)).With(planId).ServeHTTP(w, r)
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
