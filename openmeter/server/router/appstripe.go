package router

import (
	"fmt"
	"io"
	"net/http"

	appstripehttpdriver "github.com/openmeterio/openmeter/openmeter/app/stripe/httpdriver"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Handle app stripe webhook
// (POST /api/v1/apps/{appId}/stripe/webhook)
func (a *Router) AppStripeWebhook(w http.ResponseWriter, r *http.Request, appID string) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		err := fmt.Errorf("cannot read payload: %w", err)

		a.config.ErrorHandler.HandleContext(r.Context(), err)
		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w)
		return
	}

	a.appStripeHandler.AppStripeWebhook().With(appstripehttpdriver.AppStripeWebhookParams{
		AppID:   appID,
		Payload: payload,
	}).ServeHTTP(w, r)
}

// Handle update stripe api key
// (POST /api/v1/apps/{id}/stripe/api-key)
func (a *Router) UpdateStripeAPIKey(w http.ResponseWriter, r *http.Request, appID string) {
	a.appStripeHandler.UpdateStripeAPIKey().With(appID).ServeHTTP(w, r)
}

// Handle create app stripe checkout session
// (POST /api/v1/stripe/checkout/sessions)
func (a *Router) CreateStripeCheckoutSession(w http.ResponseWriter, r *http.Request) {
	a.appStripeHandler.CreateAppStripeCheckoutSession().ServeHTTP(w, r)
}
