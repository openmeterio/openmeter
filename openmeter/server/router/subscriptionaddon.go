package router

import (
	"net/http"

	httpdriver "github.com/openmeterio/openmeter/openmeter/subscription/addon/http"
)

// List subscription addons
// (GET /api/v1/subscriptions/{subscriptionId}/addons)
func (a *Router) ListSubscriptionAddons(w http.ResponseWriter, r *http.Request, subscriptionId string) {
	a.subscriptionAddonHandler.ListSubscriptionAddons().With(httpdriver.ListSubscriptionAddonsParams{
		SubscriptionID: subscriptionId,
	}).ServeHTTP(w, r)
}

// Create a subscription addon
// (POST /api/v1/subscriptions/{subscriptionId}/addons)
func (a *Router) CreateSubscriptionAddon(w http.ResponseWriter, r *http.Request, subscriptionId string) {
	a.subscriptionAddonHandler.CreateSubscriptionAddon().With(httpdriver.CreateSubscriptionAddonParams{
		SubscriptionID: subscriptionId,
	}).ServeHTTP(w, r)
}

// Get subscription addon
// (GET /api/v1/subscriptions/{subscriptionId}/addons/{subscriptionAddonId})
func (a *Router) GetSubscriptionAddon(w http.ResponseWriter, r *http.Request, subscriptionId string, subscriptionAddonId string) {
	a.subscriptionAddonHandler.GetSubscriptionAddon().With(httpdriver.GetSubscriptionAddonParams{
		SubscriptionID:      subscriptionId,
		SubscriptionAddonID: subscriptionAddonId,
	}).ServeHTTP(w, r)
}

// Update a subscription addon
// (PATCH /api/v1/subscriptions/{subscriptionId}/addons/{subscriptionAddonId})
func (a *Router) UpdateSubscriptionAddon(w http.ResponseWriter, r *http.Request, subscriptionId string, subscriptionAddonId string) {
	a.subscriptionAddonHandler.UpdateSubscriptionAddon().With(httpdriver.UpdateSubscriptionAddonParams{
		SubscriptionID:      subscriptionId,
		SubscriptionAddonID: subscriptionAddonId,
	}).ServeHTTP(w, r)
}
