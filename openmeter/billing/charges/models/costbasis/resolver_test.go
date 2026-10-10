package costbasis

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type previewResolver struct {
	state State
	input ResolveDynamicStateInput
	calls int
}

func (r *previewResolver) ResolveInitialState(context.Context, ResolveInitialStateInput) (*State, error) {
	return nil, nil
}

func (r *previewResolver) ResolveDynamicState(_ context.Context, input ResolveDynamicStateInput) (State, error) {
	r.calls++
	r.input = input

	return r.state, nil
}

func TestResolveForPreview(t *testing.T) {
	servicePeriodFrom := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	currencyID := models.NamespacedID{Namespace: "namespace", ID: "currency"}
	fiatCurrency, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	t.Run("returns persisted state", func(t *testing.T) {
		persisted := State{
			CostBasis:  alpacadecimal.NewFromFloat(0.5),
			ResolvedAt: servicePeriodFrom,
		}
		resolved, err := ResolveForPreview(t.Context(), nil, ResolveForPreviewInput{
			CurrencyID: currencyID,
			Intent: NewIntent(ManualIntent{
				FiatCurrency: fiatCurrency,
				Rate:         persisted.CostBasis,
			}),
			ResolvedCostBasis: &persisted,
			ServicePeriodFrom: servicePeriodFrom,
		})
		require.NoError(t, err)
		require.Same(t, &persisted, resolved)
	})

	t.Run("resolves dynamic state without persistence", func(t *testing.T) {
		costBasisID := "currency-cost-basis"
		state := State{
			CostBasis:   alpacadecimal.NewFromFloat(0.25),
			CostBasisID: &costBasisID,
			ResolvedAt:  servicePeriodFrom,
		}
		resolver := &previewResolver{state: state}
		intent := NewIntent(DynamicIntent{FiatCurrency: fiatCurrency})

		resolved, err := ResolveForPreview(t.Context(), resolver, ResolveForPreviewInput{
			CurrencyID:        currencyID,
			Intent:            intent,
			ServicePeriodFrom: servicePeriodFrom,
		})
		require.NoError(t, err)
		require.Equal(t, &state, resolved)
		require.Equal(t, 1, resolver.calls)
		require.Equal(t, currencyID, resolver.input.CurrencyID)
		require.Equal(t, intent, resolver.input.Intent)
		require.Equal(t, servicePeriodFrom, resolver.input.ServicePeriodFrom)
	})

	t.Run("rejects unresolved non-dynamic intent", func(t *testing.T) {
		resolver := &previewResolver{}
		_, err := ResolveForPreview(t.Context(), resolver, ResolveForPreviewInput{
			CurrencyID: currencyID,
			Intent: NewIntent(ManualIntent{
				FiatCurrency: fiatCurrency,
				Rate:         alpacadecimal.NewFromFloat(0.5),
			}),
			ServicePeriodFrom: servicePeriodFrom,
		})
		require.ErrorContains(t, err, "resolved cost basis is required")
		require.Zero(t, resolver.calls)
	})
}
