package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
)

// List notification channels
// (GET /api/v1/notification/channels)
func (a *Router) ListNotificationChannels(w http.ResponseWriter, r *http.Request, params api.ListNotificationChannelsParams) {
	a.notificationHandler.ListChannels().With(params).ServeHTTP(w, r)
}

// Create a notification channel
// (POST /api/v1/notification/channels)
func (a *Router) CreateNotificationChannel(w http.ResponseWriter, r *http.Request) {
	a.notificationHandler.CreateChannel().ServeHTTP(w, r)
}

// Delete a notification channel
// (DELETE /api/v1/notification/channels/{channelId})
func (a *Router) DeleteNotificationChannel(w http.ResponseWriter, r *http.Request, channelID string) {
	a.notificationHandler.DeleteChannel().With(channelID).ServeHTTP(w, r)
}

// Get notification channel
// (GET /api/v1/notification/channels/{channelId})
func (a *Router) GetNotificationChannel(w http.ResponseWriter, r *http.Request, channelID string) {
	a.notificationHandler.GetChannel().With(channelID).ServeHTTP(w, r)
}

// Update notification channel
// (PUT /api/v1/notification/channels/{channelId})
func (a *Router) UpdateNotificationChannel(w http.ResponseWriter, r *http.Request, channelID string) {
	a.notificationHandler.UpdateChannel().With(channelID).ServeHTTP(w, r)
}

// List notification evens
// (GET /api/v1/notification/events)
func (a *Router) ListNotificationEvents(w http.ResponseWriter, r *http.Request, params api.ListNotificationEventsParams) {
	a.notificationHandler.ListEvents().With(params).ServeHTTP(w, r)
}

// Get notification event
// (GET /api/v1/notification/events/{eventId})
func (a *Router) GetNotificationEvent(w http.ResponseWriter, r *http.Request, eventID string) {
	a.notificationHandler.GetEvent().With(eventID).ServeHTTP(w, r)
}

// Resend notification event
// (POST /api/v1/notification/events/{eventId}/resend)
func (a *Router) ResendNotificationEvent(w http.ResponseWriter, r *http.Request, eventID string) {
	a.notificationHandler.ResendEvent().With(eventID).ServeHTTP(w, r)
}

// List notification rules
// (GET /api/v1/notification/rules)
func (a *Router) ListNotificationRules(w http.ResponseWriter, r *http.Request, params api.ListNotificationRulesParams) {
	a.notificationHandler.ListRules().With(params).ServeHTTP(w, r)
}

// Create a notification rule
// (POST /api/v1/notification/rules)
func (a *Router) CreateNotificationRule(w http.ResponseWriter, r *http.Request) {
	a.notificationHandler.CreateRule().ServeHTTP(w, r)
}

// Delete a notification rule
// (DELETE /api/v1/notification/rules/{ruleId})
func (a *Router) DeleteNotificationRule(w http.ResponseWriter, r *http.Request, ruleID string) {
	a.notificationHandler.DeleteRule().With(ruleID).ServeHTTP(w, r)
}

// Get notification rule
// (GET /api/v1/notification/rules/{ruleId})
func (a *Router) GetNotificationRule(w http.ResponseWriter, r *http.Request, ruleID string) {
	a.notificationHandler.GetRule().With(ruleID).ServeHTTP(w, r)
}

// Update a notification rule
// (PUT /api/v1/notification/rules/{ruleId})
func (a *Router) UpdateNotificationRule(w http.ResponseWriter, r *http.Request, ruleID string) {
	a.notificationHandler.UpdateRule().With(ruleID).ServeHTTP(w, r)
}

// Test notification rule
// (POST /api/v1/notification/rules/{ruleId}/test)
func (a *Router) TestNotificationRule(w http.ResponseWriter, r *http.Request, ruleID string) {
	a.notificationHandler.TestRule().With(ruleID).ServeHTTP(w, r)
}
