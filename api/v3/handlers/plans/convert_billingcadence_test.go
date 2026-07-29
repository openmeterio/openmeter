package plans

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
)

func usageBasedPrice(t *testing.T, priceType string) string {
	t.Helper()

	if priceType == "unit" {
		return `{"type": "unit", "amount": "0.001"}`
	}

	return fmt.Sprintf(`{
		"type": %q,
		"tiers": [
			{"up_to_amount": "1000", "unit_price": {"type": "unit", "amount": "0.01"}},
			{"unit_price": {"type": "unit", "amount": "0.005"}}
		]
	}`, priceType)
}

func planBody(t *testing.T, price string) string {
	t.Helper()

	return fmt.Sprintf(`{
		"name": "Test Plan",
		"key": "test_plan",
		"currency": "USD",
		"billing_cadence": "P1M",
		"phases": [{
			"name": "default",
			"key": "default",
			"rate_cards": [{
				"name": "Sessions",
				"key": "sessions",
				"price": %s
			}]
		}]
	}`, price)
}

// newTestHandler builds a handler for request-decoding tests. The plan service
// is nil on purpose: these requests must fail during conversion, so a non-nil
// service would only hide a regression that let them through.
func newTestHandler() Handler {
	return New(
		func(ctx context.Context) (string, error) { return "default", nil },
		nil,
		false,
	)
}

// A usage-based rate card without billing_cadence must be rejected with a 400.
// Before this was fixed the conversion error was returned unwrapped, the v3
// error encoder did not recognise it, and the request fell through as a bare
// 500 with an empty body.
func TestCreatePlanUsageBasedRateCardMissingBillingCadence(t *testing.T) {
	for _, priceType := range []string{"unit", "graduated", "volume"} {
		t.Run(priceType, func(t *testing.T) {
			body := planBody(t, usageBasedPrice(t, priceType))

			w := httptest.NewRecorder()
			r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/plans",
				strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")

			newTestHandler().CreatePlan().ServeHTTP(w, r)

			assert.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
			assert.Contains(t, w.Body.String(), "billing_cadence")
		})
	}
}

// UpdatePlan shares the conversion path, so it must map the same failure the
// same way.
func TestUpdatePlanUsageBasedRateCardMissingBillingCadence(t *testing.T) {
	body := planBody(t, usageBasedPrice(t, "unit"))

	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/plans/01ARZ3NDEKTSV4RRFFQ69G5FAV",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	newTestHandler().UpdatePlan().With("01ARZ3NDEKTSV4RRFFQ69G5FAV").ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), "billing_cadence")
}

// Flat and free prices carry no rate-card cadence requirement, so conversion
// must keep succeeding for them.
func TestFlatRateCardWithoutBillingCadenceIsAllowed(t *testing.T) {
	var body api.CreatePlanRequest
	require.NoError(t, json.Unmarshal(
		[]byte(planBody(t, `{"type": "flat", "amount": "10"}`)), &body))

	_, err := FromAPICreatePlanRequest("default", body)
	require.NoError(t, err)
}

// The typed error has to stay identifiable through the %w wrapping applied by
// the rate card and phase converters, which is what the handler mapping relies
// on.
func TestBillingCadenceErrorSurvivesWrapping(t *testing.T) {
	var body api.CreatePlanRequest
	require.NoError(t, json.Unmarshal(
		[]byte(planBody(t, usageBasedPrice(t, "unit"))), &body))

	_, err := FromAPICreatePlanRequest("default", body)
	require.Error(t, err)

	var cadenceErr ErrRateCardBillingCadenceRequired
	require.True(t, errors.As(err, &cadenceErr))
	assert.Equal(t, "sessions", cadenceErr.Key)
}
