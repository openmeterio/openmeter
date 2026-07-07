package customerscredits

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestToAPIBillingCreditGrantPromotional(t *testing.T) {
	now := time.Date(2026, time.April, 17, 10, 0, 0, 0, time.UTC)

	charge := creditpurchase.Charge{
		ChargeBase: creditpurchase.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{
					Namespace: "ns",
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: now,
					UpdatedAt: now,
				},
				ID: "grant-1",
			},
			Intent: creditpurchase.Intent{
				Intent: meta.Intent{
					CustomerID: "cust-1",
					Currency:   currencyx.Code("USD"),
				},
				IntentMutableFields: creditpurchase.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name: "Promo credits",
					},
					CreditAmount: alpacadecimal.RequireFromString("25"),
					Settlement:   creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
				},
			},
			Status: creditpurchase.StatusActive,
		},
	}

	grant, err := toAPIBillingCreditGrant(charge)
	require.NoError(t, err)
	require.Equal(t, api.BillingCreditFundingMethodNone, grant.FundingMethod)
	require.Nil(t, grant.Purchase)
	require.Equal(t, "25", grant.Amount)
	require.Equal(t, api.BillingCreditGrantStatusActive, grant.Status)
	require.Nil(t, grant.VoidedAt)

	t.Run("ledger-derived voiding overrides the charge status", func(t *testing.T) {
		voidedAt := now.Add(time.Hour)

		voidedCharge := charge
		voidedCharge.State.VoidedAt = &voidedAt

		grant, err := toAPIBillingCreditGrant(voidedCharge)
		require.NoError(t, err)
		require.Equal(t, api.BillingCreditGrantStatusVoided, grant.Status)
		require.Equal(t, lo.ToPtr(voidedAt), grant.VoidedAt)
	})
}

func TestToAPIBillingCreditGrantStatusUsesExpiry(t *testing.T) {
	now := time.Date(2026, time.April, 17, 10, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	charge := creditpurchase.Charge{
		ChargeBase: creditpurchase.ChargeBase{
			Intent: creditpurchase.Intent{
				Intent: meta.Intent{
					CustomerID: "cust-1",
					Currency:   currencyx.Code("USD"),
				},
				IntentMutableFields: creditpurchase.IntentMutableFields{
					CreditAmount: alpacadecimal.RequireFromString("25"),
					Settlement:   creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
				},
			},
			Status: creditpurchase.StatusActive,
		},
	}

	t.Run("past expiry is public expired", func(t *testing.T) {
		expiredAt := now.Add(-time.Nanosecond)
		expiredCharge := charge
		expiredCharge.Intent.ExpiresAt = &expiredAt

		grant, err := toAPIBillingCreditGrant(expiredCharge)
		require.NoError(t, err)
		require.Equal(t, api.BillingCreditGrantStatusExpired, grant.Status)
	})

	t.Run("expiry at now is public expired", func(t *testing.T) {
		expiredCharge := charge
		expiredCharge.Intent.ExpiresAt = &now

		grant, err := toAPIBillingCreditGrant(expiredCharge)
		require.NoError(t, err)
		require.Equal(t, api.BillingCreditGrantStatusExpired, grant.Status)
	})

	t.Run("future expiry stays active", func(t *testing.T) {
		expiresAt := now.Add(time.Nanosecond)
		activeCharge := charge
		activeCharge.Intent.ExpiresAt = &expiresAt

		grant, err := toAPIBillingCreditGrant(activeCharge)
		require.NoError(t, err)
		require.Equal(t, api.BillingCreditGrantStatusActive, grant.Status)
	})

	t.Run("voided wins over expired", func(t *testing.T) {
		expiredAt := now.Add(-time.Nanosecond)
		voidedAt := now
		voidedCharge := charge
		voidedCharge.Intent.ExpiresAt = &expiredAt
		voidedCharge.State.VoidedAt = &voidedAt

		grant, err := toAPIBillingCreditGrant(voidedCharge)
		require.NoError(t, err)
		require.Equal(t, api.BillingCreditGrantStatusVoided, grant.Status)
	})
}

func TestToAPIBillingCreditGrantKey(t *testing.T) {
	now := time.Date(2026, time.April, 17, 10, 0, 0, 0, time.UTC)

	newCharge := func(key *string) creditpurchase.Charge {
		return creditpurchase.Charge{
			ChargeBase: creditpurchase.ChargeBase{
				ManagedResource: meta.ManagedResource{
					NamespacedModel: models.NamespacedModel{
						Namespace: "ns",
					},
					ManagedModel: models.ManagedModel{
						CreatedAt: now,
						UpdatedAt: now,
					},
					ID: "grant-1",
				},
				Intent: creditpurchase.Intent{
					Intent: meta.Intent{
						CustomerID: "cust-1",
						Currency:   currencyx.Code("USD"),
					},
					IntentMutableFields: creditpurchase.IntentMutableFields{
						IntentMutableFields: meta.IntentMutableFields{
							Name: "Promo credits",
						},
						CreditAmount: alpacadecimal.RequireFromString("25"),
						Settlement:   creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
					},
					Key: key,
				},
				Status: creditpurchase.StatusActive,
			},
		}
	}

	t.Run("maps the idempotency key into the read response", func(t *testing.T) {
		grant, err := toAPIBillingCreditGrant(newCharge(lo.ToPtr("welcome-bonus")))
		require.NoError(t, err)
		require.Equal(t, lo.ToPtr("welcome-bonus"), grant.Key)
	})

	t.Run("omits the key when the grant has none", func(t *testing.T) {
		grant, err := toAPIBillingCreditGrant(newCharge(nil))
		require.NoError(t, err)
		require.Nil(t, grant.Key)
	})
}
