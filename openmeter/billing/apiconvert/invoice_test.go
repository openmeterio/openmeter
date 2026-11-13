package apiconvert

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestMapTotalsToAPIIncludesCreditsTotal(t *testing.T) {
	got := mapTotalsToAPI(totals.Totals{
		Amount:              alpacadecimal.NewFromInt(100),
		ChargesTotal:        alpacadecimal.NewFromInt(0),
		DiscountsTotal:      alpacadecimal.NewFromInt(10),
		CreditsTotal:        alpacadecimal.NewFromInt(15),
		TaxesInclusiveTotal: alpacadecimal.NewFromInt(0),
		TaxesExclusiveTotal: alpacadecimal.NewFromInt(5),
		TaxesTotal:          alpacadecimal.NewFromInt(5),
		Total:               alpacadecimal.NewFromInt(80),
	})

	require.Equal(t, "15", got.CreditsTotal)
}

func TestMapInvoiceLineCreditsToAPI(t *testing.T) {
	got := mapInvoiceLineCreditsToAPI(billing.CreditsApplied{
		{
			Amount:              alpacadecimal.NewFromFloat(12.34),
			Description:         "credit grant",
			CreditRealizationID: "01K8N36X000000000000000001",
		},
		{
			Amount:              alpacadecimal.NewFromInt(5),
			CreditRealizationID: "01K8N36X000000000000000002",
		},
	})

	require.NotNil(t, got)
	require.Len(t, *got, 2)
	require.Equal(t, "12.34", (*got)[0].Amount)
	require.Equal(t, "credit grant", *(*got)[0].Description)
	require.Equal(t, "5", (*got)[1].Amount)
	require.Nil(t, (*got)[1].Description)
}

func TestMapInvoiceLineCreditsToAPIOmitsEmptyCredits(t *testing.T) {
	require.Nil(t, mapInvoiceLineCreditsToAPI(nil))
}

func TestMapSubscriptionReferencesToAPIKeepsIdsOnDocumentedFields(t *testing.T) {
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)

	got := ToAPIInvoiceLineSubscriptionReference(&billing.SubscriptionReference{
		SubscriptionID: "01M6QNHXFHPA5QN3TGVPYFHZPR",
		PhaseID:        "01PHASE0000000000000000001",
		ItemID:         "01ITEM00000000000000000001",
		BillingPeriod: timeutil.ClosedPeriod{
			From: from,
			To:   to,
		},
	})

	require.NotNil(t, got)
	require.Equal(t, "01M6QNHXFHPA5QN3TGVPYFHZPR", got.Subscription.Id)
	require.Equal(t, "01PHASE0000000000000000001", got.Phase.Id)
	require.Equal(t, "01ITEM00000000000000000001", got.Item.Id)
	require.True(t, from.Equal(got.BillingPeriod.From))
	require.True(t, to.Equal(got.BillingPeriod.To))
	require.Nil(t, ToAPIInvoiceLineSubscriptionReference(nil))
}
