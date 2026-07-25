package breakage

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestListExpiredBreakageImpactsGroupsBySourceChargeID(t *testing.T) {
	expiresAt := time.Date(2026, 4, 11, 9, 0, 0, 0, time.UTC)
	customerID := customer.CustomerID{
		Namespace: "ns",
		ID:        "customer-id",
	}
	currency := currencyx.Code("USD")
	chargeA := "charge-a"
	chargeB := "charge-b"
	planAID := "01KBREAKAGE000000000000000001"

	svc := &service{
		adapter: fakeBreakageAdapter{
			expiredRecords: []Record{
				{
					ID:             models.NamespacedID{Namespace: "ns", ID: planAID},
					Kind:           ledger.BreakageKindPlan,
					Amount:         alpacadecimal.NewFromInt(10),
					CustomerID:     customerID,
					Currency:       currency,
					ExpiresAt:      expiresAt,
					SourceKind:     SourceKindCreditPurchase,
					SourceChargeID: &chargeA,
				},
				{
					ID:             models.NamespacedID{Namespace: "ns", ID: "01KBREAKAGE000000000000000002"},
					Kind:           ledger.BreakageKindRelease,
					Amount:         alpacadecimal.NewFromInt(3),
					CustomerID:     customerID,
					Currency:       currency,
					ExpiresAt:      expiresAt,
					SourceKind:     SourceKindUsage,
					SourceChargeID: &chargeA,
					PlanID:         &planAID,
				},
				{
					ID:             models.NamespacedID{Namespace: "ns", ID: "01KBREAKAGE000000000000000003"},
					Kind:           ledger.BreakageKindPlan,
					Amount:         alpacadecimal.NewFromInt(7),
					CustomerID:     customerID,
					Currency:       currency,
					ExpiresAt:      expiresAt,
					SourceKind:     SourceKindCreditPurchase,
					SourceChargeID: &chargeB,
				},
			},
		},
	}

	got, err := svc.ListExpiredBreakageImpacts(t.Context(), ListExpiredBreakageImpactsInput{
		CustomerID: customerID,
		Currency:   &currency,
		AsOf:       expiresAt,
		Limit:      20,
	})
	require.NoError(t, err)
	require.Len(t, got.Items, 2)

	amountByChargeID := make(map[string]float64, len(got.Items))
	for _, item := range got.Items {
		require.Equal(t, ledger.CollectionTypeBreakage, item.Annotations[ledger.AnnotationCollectionType])

		chargeID, ok := item.Annotations[ledger.AnnotationChargeID].(string)
		require.True(t, ok)
		amountByChargeID[chargeID] = item.Amount.InexactFloat64()
	}

	require.Equal(t, float64(-7), amountByChargeID[chargeA])
	require.Equal(t, float64(-7), amountByChargeID[chargeB])
}

type fakeBreakageAdapter struct {
	expiredRecords []Record
}

func (a fakeBreakageAdapter) CreateRecords(context.Context, CreateRecordsInput) error {
	return nil
}

func (a fakeBreakageAdapter) ListCandidateRecords(context.Context, ListPlansInput) ([]Record, error) {
	return nil, nil
}

func (a fakeBreakageAdapter) ListReleaseRecords(context.Context, ListReleasesInput) ([]Record, error) {
	return nil, nil
}

func (a fakeBreakageAdapter) ListExpiredRecords(context.Context, ListExpiredRecordsInput) ([]Record, error) {
	return a.expiredRecords, nil
}
