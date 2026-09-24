package flatfee

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestIntentEqual(t *testing.T) {
	fiatCurrency := currenciestestutils.NewFiatCurrency(t, "USD")
	base := newValidIntent(t, fiatCurrency, productcatalog.CreditOnlySettlementMode)

	t.Run("equal intent", func(t *testing.T) {
		other := base
		require.True(t, base.Equal(other))
	})

	t.Run("different amount", func(t *testing.T) {
		other := base
		other.AmountBeforeProration = alpacadecimal.NewFromInt(200)
		require.False(t, base.Equal(other))
	})

	t.Run("different period", func(t *testing.T) {
		other := base
		other.ServicePeriod.To = other.ServicePeriod.To.AddDate(0, 1, 0)
		require.False(t, base.Equal(other))
	})

	t.Run("equal deletion instant in different time zones", func(t *testing.T) {
		first := base
		second := base
		instant := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
		local := instant.In(time.FixedZone("offset", 3600))
		first.IntentDeletedAt = &instant
		second.IntentDeletedAt = &local
		require.True(t, first.Equal(second))
	})

	t.Run("currency expansion does not change intent", func(t *testing.T) {
		other := base
		other.Currency.CostBasis = &[]currencies.CostBasis{}
		require.True(t, base.Equal(other))
	})

	t.Run("different custom currency IDs", func(t *testing.T) {
		namespace := ulid.Make().String()
		first := newValidIntent(t, newCustomCurrency(t, namespace), productcatalog.CreditOnlySettlementMode)
		second := first
		second.Currency = newCustomCurrency(t, namespace)
		require.False(t, first.Equal(second))
	})
}

func TestIntentEqualCostBasis(t *testing.T) {
	currency := newCustomCurrency(t, ulid.Make().String())
	base := newValidIntent(t, currency, productcatalog.CreditThenInvoiceSettlementMode)
	baseCostBasis := newManualCostBasisIntent(t)
	base.CostBasis = &baseCostBasis

	other := base
	otherCostBasis := baseCostBasis.Clone()
	other.CostBasis = &otherCostBasis
	require.True(t, base.Equal(other))

	otherCostBasis = costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: newFiatCurrency(t, "USD"),
		Rate:         alpacadecimal.NewFromInt(3),
	})
	require.False(t, base.Equal(other))
}
