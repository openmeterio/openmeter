package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
	subscriptionhttpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/http"
)

// (POST /api/v1/subscriptions)
func (a *Router) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	a.subscriptionHandler.CreateSubscription().ServeHTTP(w, r)
}

func (a *Router) ChangeSubscription(w http.ResponseWriter, r *http.Request, subscriptionId string) {
	a.subscriptionHandler.ChangeSubscription().With(subscriptionhttpdriver.ChangeSubscriptionParams{
		ID: subscriptionId,
	}).ServeHTTP(w, r)
}

// (GET /api/v1/subscriptions/{subscriptionId})
func (a *Router) GetSubscription(w http.ResponseWriter, r *http.Request, subscriptionId string, params api.GetSubscriptionParams) {
	a.subscriptionHandler.GetSubscription().With(subscriptionhttpdriver.GetSubscriptionParams{
		Query: params,
		ID:    subscriptionId,
	}).ServeHTTP(w, r)
}

// (PATCH /api/v1/subscriptions/{subscriptionId})
func (a *Router) EditSubscription(w http.ResponseWriter, r *http.Request, subscriptionId string) {
	a.subscriptionHandler.EditSubscription().With(subscriptionhttpdriver.EditSubscriptionParams{
		ID: subscriptionId,
	}).ServeHTTP(w, r)
}

// (POST /api/v1/subscriptions/{subscriptionId}/cancel)
func (a *Router) CancelSubscription(w http.ResponseWriter, r *http.Request, subscriptionId string) {
	a.subscriptionHandler.CancelSubscription().With(subscriptionhttpdriver.CancelSubscriptionParams{
		ID: subscriptionId,
	}).ServeHTTP(w, r)
}

// (POST /api/v1/subscriptions/{subscriptionId}/migrate)
func (a *Router) MigrateSubscription(w http.ResponseWriter, r *http.Request, subscriptionId string) {
	a.subscriptionHandler.MigrateSubscription().With(subscriptionhttpdriver.MigrateSubscriptionParams{
		ID: subscriptionId,
	}).ServeHTTP(w, r)
}

// (POST /api/v1/subscriptions/{subscriptionId}/unschedule-cancelation)
func (a *Router) UnscheduleCancelation(w http.ResponseWriter, r *http.Request, subscriptionId string) {
	a.subscriptionHandler.ContinueSubscription().With(subscriptionhttpdriver.ContinueSubscriptionParams{
		ID: subscriptionId,
	}).ServeHTTP(w, r)
}

// (POST /api/v1/subscriptions/{subscriptionId}/restore)
func (a *Router) RestoreSubscription(w http.ResponseWriter, r *http.Request, subscriptionId string) {
	a.subscriptionHandler.RestoreSubscription().With(subscriptionhttpdriver.RestoreSubscriptionParams{
		ID: subscriptionId,
	}).ServeHTTP(w, r)
}
