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
