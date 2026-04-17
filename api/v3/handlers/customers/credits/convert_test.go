package customerscredits

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
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
					Name:       "Promo credits",
					CustomerID: "cust-1",
					Currency:   currencyx.Code("USD"),
				},
				CreditAmount: alpacadecimal.RequireFromString("25"),
				Settlement:   creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
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
