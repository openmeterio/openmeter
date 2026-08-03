package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/pkg/clock"
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
