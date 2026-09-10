package subscriptions

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

// TestSubscriptionGenericErrorEncoder verifies the shared encoder maps the
// models.Generic* errors the subscription command/workflow services return to
// their HTTP statuses, including when wrapped. These types are plain error
// wrappers (not ValidationIssues), so the plain v3 generic encoder does not
// classify them and would 500 ordinary client mistakes — this encoder is what
// keeps unschedule/restore/create returning correct 4xx codes.
func TestSubscriptionGenericErrorEncoder(t *testing.T) {
	enc := subscriptionGenericErrorEncoder()

	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"validation -> 400", models.NewGenericValidationError(errors.New("bad")), http.StatusBadRequest},
		{"forbidden -> 403", models.NewGenericForbiddenError(errors.New("not allowed")), http.StatusForbidden},
		{"conflict -> 409", models.NewGenericConflictError(errors.New("conflict")), http.StatusConflict},
		{"not found -> 404", models.NewGenericNotFoundError(errors.New("missing")), http.StatusNotFound},
		// Wrapped errors still classify (HandleErrorIfTypeMatches uses errors.As):
		// the delete state-machine guard and the repo return wrapped Generic*
		// errors in practice.
		{
			"wrapped forbidden -> 403",
			fmt.Errorf("delete: %w", models.NewGenericForbiddenError(errors.New("scheduled only"))),
			http.StatusForbidden,
		},
		{"subscription-not-found -> 404", subscription.NewSubscriptionNotFoundError("sub_1"), http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", nil)

			handled := enc(t.Context(), tt.err, w, r)

			require.True(t, handled, "encoder should classify the error")
			require.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

// TestSubscriptionGenericErrorEncoderDefersUnknown documents the boundary: an
// error that is not a mapped Generic* type is not handled here, so it falls
// through to the framework's default (500) rather than being silently swallowed.
func TestSubscriptionGenericErrorEncoderDefersUnknown(t *testing.T) {
	enc := subscriptionGenericErrorEncoder()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)

	handled := enc(t.Context(), errors.New("some unexpected error"), w, r)

	require.False(t, handled, "an unmapped error should not be handled by this encoder")
}
