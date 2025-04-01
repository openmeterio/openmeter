package router

import (
	"net/http"
)

// List subscription addons
// (GET /api/v1/subscriptions/{subscriptionId}/addons)
func (a *Router) ListSubscriptionAddons(w http.ResponseWriter, r *http.Request, subscriptionId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Create a subscription addon
// (POST /api/v1/subscriptions/{subscriptionId}/addons)
func (a *Router) CreateSubscriptionAddon(w http.ResponseWriter, r *http.Request, subscriptionId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Get subscription addon
// (GET /api/v1/subscriptions/{subscriptionId}/addons/{subscriptionAddonId})
func (a *Router) GetSubscriptionAddon(w http.ResponseWriter, r *http.Request, subscriptionId string, subscriptionAddonId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Cancel subscription addon
// (POST /api/v1/subscriptions/{subscriptionId}/addons/{subscriptionAddonId}/cancel)
func (a *Router) CancelSubscriptionAddon(w http.ResponseWriter, r *http.Request, subscriptionId string, subscriptionAddonId string) {
	w.WriteHeader(http.StatusNotImplemented)
}
