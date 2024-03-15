// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
)

// List notification channels
// (GET /api/v1/notification/channels)
func (a *Router) ListNotificationChannels(w http.ResponseWriter, r *http.Request, params api.ListNotificationChannelsParams) {
	unimplemented.ListNotificationChannels(w, r, params)
}

// Create a notification channel
// (POST /api/v1/notification/channels)
func (a *Router) CreateNotificationChannel(w http.ResponseWriter, r *http.Request) {
	unimplemented.CreateNotificationChannel(w, r)
}

// Delete a notification channel
// (DELETE /api/v1/notification/channels/{channelId})
func (a *Router) DeleteNotificationChannel(w http.ResponseWriter, r *http.Request, channelID api.ChannelId) {
	unimplemented.DeleteNotificationChannel(w, r, channelID)
}

// Get notification channel
// (GET /api/v1/notification/channels/{channelId})
func (a *Router) GetNotificationChannel(w http.ResponseWriter, r *http.Request, channelID api.ChannelId) {
	unimplemented.GetNotificationChannel(w, r, channelID)
}

// Update notification channel
// (PUT /api/v1/notification/channels/{channelId})
func (a *Router) UpdateNotificationChannel(w http.ResponseWriter, r *http.Request, channelID api.ChannelId) {
	unimplemented.UpdateNotificationChannel(w, r, channelID)
}

// List notification evens
// (GET /api/v1/notification/events)
func (a *Router) ListNotificationEvents(w http.ResponseWriter, r *http.Request, params api.ListNotificationEventsParams) {
	unimplemented.ListNotificationEvents(w, r, params)
}

// Get notification event
// (GET /api/v1/notification/events/{eventId})
func (a *Router) GetNotificationEvent(w http.ResponseWriter, r *http.Request, eventID api.EventId) {
	unimplemented.GetNotificationEvent(w, r, eventID)
}

// List notification rules
// (GET /api/v1/notification/rules)
func (a *Router) ListNotificationRules(w http.ResponseWriter, r *http.Request, params api.ListNotificationRulesParams) {
	unimplemented.ListNotificationRules(w, r, params)
}

// Create a notification rule
// (POST /api/v1/notification/rules)
func (a *Router) CreateNotificationRule(w http.ResponseWriter, r *http.Request) {
	unimplemented.CreateNotificationRule(w, r)
}

// Delete a notification rule
// (DELETE /api/v1/notification/rules/{ruleId})
func (a *Router) DeleteNotificationRule(w http.ResponseWriter, r *http.Request, ruleID api.RuleId) {
	unimplemented.DeleteNotificationRule(w, r, ruleID)
}

// Get notification rule
// (GET /api/v1/notification/rules/{ruleId})
func (a *Router) GetNotificationRule(w http.ResponseWriter, r *http.Request, ruleID api.RuleId) {
	unimplemented.GetNotificationRule(w, r, ruleID)
}

// Update a notification rule
// (PUT /api/v1/notification/rules/{ruleId})
func (a *Router) UpdateNotificationRule(w http.ResponseWriter, r *http.Request, ruleID api.RuleId) {
	unimplemented.UpdateNotificationRule(w, r, ruleID)
}
