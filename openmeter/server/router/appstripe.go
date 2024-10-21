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
	if !a.config.AppsEnabled {
		models.NewStatusProblem(r.Context(), fmt.Errorf("apps are disabled"), http.StatusNotImplemented).Respond(w)
		return
	}
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

// Handle create app stripe checkout session
// (POST /api/v1/integration/stripe/checkout/sessions)
func (a *Router) CreateStripeCheckoutSession(w http.ResponseWriter, r *http.Request) {
	if !a.config.AppsEnabled {
		unimplemented.CreateStripeCheckoutSession(w, r)
		return
	}

	a.appStripeHandler.CreateAppStripeCheckoutSession().ServeHTTP(w, r)
}
