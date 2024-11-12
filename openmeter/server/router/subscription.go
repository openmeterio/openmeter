package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
)

// (POST /api/v1/subscriptions)
func (a *Router) SubscriptionsCreate(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (GET /api/v1/subscriptions/{subscriptionId})
func (a *Router) GetSubscription(w http.ResponseWriter, r *http.Request, subscriptionId string, params api.GetSubscriptionParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (PATCH /api/v1/subscriptions/{subscriptionId})
func (a *Router) EditSubscription(w http.ResponseWriter, r *http.Request, subscriptionId string) {
	w.WriteHeader(http.StatusNotImplemented)
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
