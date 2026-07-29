package plans

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
)

// A usage-based rate card without billing_cadence must be rejected by the
// converter, and that rejection must reach the client as a 400 rather than
// falling through the error encoder as an unhandled 500.
func TestUsageBasedRateCardMissingBillingCadence(t *testing.T) {
	for _, priceType := range []string{"unit", "graduated", "volume"} {
		t.Run(priceType, func(t *testing.T) {
			var price string
			switch priceType {
			case "unit":
				price = `{"type": "unit", "amount": "0.001"}`
			default:
				price = fmt.Sprintf(`{
					"type": %q,
					"tiers": [
						{"up_to_amount": "1000", "unit_price": {"type": "unit", "amount": "0.01"}},
						{"unit_price": {"type": "unit", "amount": "0.005"}}
					]
				}`, priceType)
			}

			var body api.CreatePlanRequest
			require.NoError(t, json.Unmarshal([]byte(fmt.Sprintf(`{
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
			}`, price)), &body))

			_, err := FromAPICreatePlanRequest("default", body)
			require.Error(t, err)

			// The error is wrapped by the phase and rate card converters, so it
			// has to remain identifiable through the wrapping chain.
			var cadenceErr ErrRateCardBillingCadenceRequired
			require.True(t, errors.As(err, &cadenceErr))
			assert.Equal(t, "sessions", cadenceErr.Key)

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/plans", nil)
			handled := apierrors.GenericErrorEncoder()(
				context.Background(),
				asBadRequestIfConversionError(context.Background(), err),
				w, r,
			)

			require.True(t, handled, "error encoder must handle the error, otherwise it surfaces as a bare 500")
			assert.Equal(t, 400, w.Code)
			assert.Contains(t, w.Body.String(), "billing_cadence")
		})
	}
}

// Flat and free prices have no rate-card billing cadence requirement, so they
// must keep converting cleanly.
func TestFlatRateCardWithoutBillingCadenceIsAllowed(t *testing.T) {
	var body api.CreatePlanRequest
	require.NoError(t, json.Unmarshal([]byte(`{
		"name": "Test Plan",
		"key": "test_plan",
		"currency": "USD",
		"billing_cadence": "P1M",
		"phases": [{
			"name": "default",
			"key": "default",
			"rate_cards": [{
				"name": "Base",
				"key": "base",
				"price": {"type": "flat", "amount": "10"}
			}]
		}]
	}`), &body))

	_, err := FromAPICreatePlanRequest("default", body)
	require.NoError(t, err)
}

// Errors that are not conversion failures must pass through untouched, so
// unrelated failures keep their own status mapping.
func TestUnrelatedErrorsArePassedThrough(t *testing.T) {
	sentinel := errors.New("some other failure")
	assert.Equal(t, sentinel, asBadRequestIfConversionError(context.Background(), sentinel))
	assert.NoError(t, asBadRequestIfConversionError(context.Background(), nil))
}
