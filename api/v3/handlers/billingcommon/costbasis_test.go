package billingcommon

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestFromAPIChargeCostBasis(t *testing.T) {
	t.Run("nil is omitted", func(t *testing.T) {
		out, err := FromAPIChargeCostBasis(nil)
		require.NoError(t, err)
		require.Nil(t, out)
	})

	t.Run("dynamic", func(t *testing.T) {
		var in api.BillingChargeCostBasis
		require.NoError(t, in.FromBillingChargeCostBasisDynamic(api.BillingChargeCostBasisDynamic{
			Type:         api.BillingChargeCostBasisDynamicTypeDynamic,
			FiatCurrency: "USD",
		}))

		out, err := FromAPIChargeCostBasis(&in)
		require.NoError(t, err)
		require.NotNil(t, out)
		assert.Equal(t, costbasis.ModeDynamic, out.Kind())

		fiat, err := out.GetFiatCurrency()
		require.NoError(t, err)
		assert.Equal(t, currencyx.Code("USD"), fiat.Details().Code)
	})

	t.Run("manual", func(t *testing.T) {
		var in api.BillingChargeCostBasis
		require.NoError(t, in.FromBillingChargeCostBasisManual(api.BillingChargeCostBasisManual{
			Type:         api.BillingChargeCostBasisManualTypeManual,
			FiatCurrency: lo.ToPtr("USD"),
			Rate:         "0.5",
		}))

		out, err := FromAPIChargeCostBasis(&in)
		require.NoError(t, err)
		require.NotNil(t, out)

		manual, err := out.AsManual()
		require.NoError(t, err)
		assert.Equal(t, 0.5, manual.Rate.InexactFloat64())
	})

	t.Run("pinned", func(t *testing.T) {
		var in api.BillingChargeCostBasis
		require.NoError(t, in.FromBillingChargeCostBasisPinned(api.BillingChargeCostBasisPinned{
			Type:         api.BillingChargeCostBasisPinnedTypePinned,
			FiatCurrency: "USD",
			CostBasisId:  "cb-1",
		}))

		out, err := FromAPIChargeCostBasis(&in)
		require.NoError(t, err)
		require.NotNil(t, out)

		pinned, err := out.AsPinned()
		require.NoError(t, err)
		assert.Equal(t, "cb-1", pinned.CurrencyCostBasisID)
	})

	t.Run("invalid rate", func(t *testing.T) {
		var in api.BillingChargeCostBasis
		require.NoError(t, in.FromBillingChargeCostBasisManual(api.BillingChargeCostBasisManual{
			Type:         api.BillingChargeCostBasisManualTypeManual,
			FiatCurrency: lo.ToPtr("USD"),
			Rate:         "not-a-number",
		}))

		_, err := FromAPIChargeCostBasis(&in)
		require.Error(t, err)
	})
}

func TestToAPIChargeResolvedCostBasis(t *testing.T) {
	fiat, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	resolvedAt := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	state := &costbasis.State{
		CostBasis:   alpacadecimal.NewFromFloat(0.5),
		CostBasisID: lo.ToPtr("cb-1"),
		ResolvedAt:  resolvedAt,
	}

	t.Run("nil state is omitted", func(t *testing.T) {
		out, err := ToAPIChargeResolvedCostBasis(fiat, nil)
		require.NoError(t, err)
		require.Nil(t, out)
	})

	t.Run("state without fiat currency is rejected", func(t *testing.T) {
		_, err := ToAPIChargeResolvedCostBasis(nil, state)
		require.Error(t, err)
	})

	t.Run("resolved", func(t *testing.T) {
		out, err := ToAPIChargeResolvedCostBasis(fiat, state)
		require.NoError(t, err)
		require.NotNil(t, out)
		assert.Equal(t, api.CurrencyCode("USD"), out.FiatCurrency)
		assert.Equal(t, "0.5", out.Rate)
		assert.Equal(t, lo.ToPtr("cb-1"), out.CostBasisId)
		assert.Equal(t, resolvedAt, out.ResolvedAt)
	})
}
