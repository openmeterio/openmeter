package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// The charge fixtures are hand-assembled because some rejected lifecycle
// states (notably a still-created charge) cannot be produced through the
// public grant creation flows today, but the guard must stay defensive.
func TestValidateChargeVoidable(t *testing.T) {
	now := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	newCharge := func(status creditpurchase.Status, mutate func(*creditpurchase.Charge)) creditpurchase.Charge {
		charge := creditpurchase.Charge{
			ChargeBase: creditpurchase.ChargeBase{
				Status: status,
			},
		}
		charge.ID = "grant-1"
		if mutate != nil {
			mutate(&charge)
		}

		return charge
	}

	t.Run("active charge is voidable", func(t *testing.T) {
		require.NoError(t, validateChargeVoidable(newCharge(creditpurchase.StatusActive, nil)))
	})

	t.Run("final charge is voidable", func(t *testing.T) {
		require.NoError(t, validateChargeVoidable(newCharge(creditpurchase.StatusFinal, nil)))
	})

	t.Run("detailed active payment state is voidable", func(t *testing.T) {
		require.NoError(t, validateChargeVoidable(newCharge(creditpurchase.StatusActivePaymentPending, nil)))
	})

	t.Run("created (pending) charge is rejected with conflict", func(t *testing.T) {
		err := validateChargeVoidable(newCharge(creditpurchase.StatusCreated, nil))
		require.Error(t, err)
		require.True(t, models.IsGenericConflictError(err))
	})

	t.Run("deleted charge is rejected as not found", func(t *testing.T) {
		err := validateChargeVoidable(newCharge(creditpurchase.StatusDeleted, func(charge *creditpurchase.Charge) {
			charge.DeletedAt = &now
		}))
		require.Error(t, err)
		require.True(t, models.IsGenericNotFoundError(err))
	})

	t.Run("already expired charge is rejected with conflict", func(t *testing.T) {
		expiresAt := now.Add(-time.Hour)
		err := validateChargeVoidable(newCharge(creditpurchase.StatusActive, func(charge *creditpurchase.Charge) {
			charge.Intent.ExpiresAt = &expiresAt
		}))
		require.Error(t, err)
		require.True(t, models.IsGenericConflictError(err))
	})

	t.Run("future expiry stays voidable", func(t *testing.T) {
		expiresAt := now.Add(time.Hour)
		require.NoError(t, validateChargeVoidable(newCharge(creditpurchase.StatusActive, func(charge *creditpurchase.Charge) {
			charge.Intent.ExpiresAt = &expiresAt
		})))
	})
}

func TestToCostBasis(t *testing.T) {
	fiatCurrency := currenciestestutils.NewFiatCurrency(t, "USD")
	customCurrency := currenciestestutils.NewCustomCurrency(t, "TOKENS", 3)

	usd, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	requireFiatRate := func(t *testing.T, costBasis creditpurchase.CostBasis, want float64) {
		t.Helper()

		require.Equal(t, creditpurchase.CostBasisTypeFiat, costBasis.Type())

		fiat, err := costBasis.AsFiat()
		require.NoError(t, err)
		require.Equal(t, want, fiat.Rate.InexactFloat64())
	}

	t.Run("fiat grant defaults to a rate of one", func(t *testing.T) {
		costBasis, err := toCostBasis(&creditgrant.PurchaseTerms{Currency: "USD"}, fiatCurrency)
		require.NoError(t, err)
		requireFiatRate(t, costBasis, 1)
	})

	t.Run("fiat grant uses the deprecated per unit rate", func(t *testing.T) {
		costBasis, err := toCostBasis(&creditgrant.PurchaseTerms{
			Currency:         "USD",
			PerUnitCostBasis: lo.ToPtr(alpacadecimal.NewFromFloat(0.25)),
		}, fiatCurrency)
		require.NoError(t, err)
		requireFiatRate(t, costBasis, 0.25)
	})

	t.Run("fiat grant keeps the fiat rate cost basis", func(t *testing.T) {
		costBasis, err := toCostBasis(&creditgrant.PurchaseTerms{
			Currency:  "USD",
			CostBasis: lo.ToPtr(creditpurchase.NewCostBasis(creditpurchase.FiatCostBasis{Rate: alpacadecimal.NewFromFloat(0.5)})),
		}, fiatCurrency)
		require.NoError(t, err)
		requireFiatRate(t, costBasis, 0.5)
	})

	t.Run("custom grant keeps the intent", func(t *testing.T) {
		costBasis, err := toCostBasis(&creditgrant.PurchaseTerms{
			Currency:  "USD",
			CostBasis: lo.ToPtr(creditpurchase.NewCostBasis(costbasis.NewIntent(costbasis.DynamicIntent{FiatCurrency: usd}))),
		}, customCurrency)
		require.NoError(t, err)
		require.Equal(t, creditpurchase.CostBasisTypeCustomCurrency, costBasis.Type())
		require.Equal(t, costbasis.ModeDynamic, costBasis.GetCustomCurrencyModeOrEmpty())
	})
}
