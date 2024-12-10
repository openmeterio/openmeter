package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
	subscriptionhttpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/http"
)

// (POST /api/v1/subscriptions)
func (a *Router) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	if !a.config.ProductCatalogEnabled {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	a.subscriptionHandler.CreateSubscription().ServeHTTP(w, r)
}

func (a *Router) ChangeSubscription(w http.ResponseWriter, r *http.Request, subscriptionId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (GET /api/v1/subscriptions/{subscriptionId})
func (a *Router) GetSubscription(w http.ResponseWriter, r *http.Request, subscriptionId string, params api.GetSubscriptionParams) {
	if !a.config.ProductCatalogEnabled {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	a.subscriptionHandler.GetSubscription().With(subscriptionhttpdriver.GetSubscriptionParams{
		Query: params,
		ID:    subscriptionId,
	}).ServeHTTP(w, r)
}

// (PATCH /api/v1/subscriptions/{subscriptionId})
func (a *Router) EditSubscription(w http.ResponseWriter, r *http.Request, subscriptionId string) {
	if !a.config.ProductCatalogEnabled {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	a.subscriptionHandler.EditSubscription().With(subscriptionhttpdriver.EditSubscriptionParams{
		ID: subscriptionId,
	}).ServeHTTP(w, r)
}

// (POST /api/v1/subscriptions/{subscriptionId}/cancel)
func (a *Router) CancelSubscription(w http.ResponseWriter, r *http.Request, subscriptionId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (POST /api/v1/subscriptions/{subscriptionId}/migrate)
func (a *Router) MigrateSubscription(w http.ResponseWriter, r *http.Request, subscriptionId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (POST /api/v1/subscriptions/{subscriptionId}/unschedule-cancelation)
func (a *Router) UnscheduleCancelation(w http.ResponseWriter, r *http.Request, subscriptionId string) {
	w.WriteHeader(http.StatusNotImplemented)
}
