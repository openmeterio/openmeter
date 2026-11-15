package persistedstate

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type mockBillingService struct {
	standardLines   billing.StandardLines
	gatheringLines  billing.GatheringLines
	splitLineGroups []billing.SplitLineHierarchy
	invoices        billing.ListInvoicesResponse
}

func (m *mockBillingService) GetStandardLinesForSubscription(_ context.Context, _ billing.GetLinesForSubscriptionInput) (billing.StandardLines, error) {
	return m.standardLines, nil
}

func (m *mockBillingService) GetGatheringLinesForSubscription(_ context.Context, _ billing.GetLinesForSubscriptionInput) (billing.GatheringLines, error) {
	return m.gatheringLines, nil
}

func (m *mockBillingService) GetSplitLineGroupsForSubscription(_ context.Context, _ billing.GetLinesForSubscriptionInput) ([]billing.SplitLineHierarchy, error) {
	return m.splitLineGroups, nil
}

func (m *mockBillingService) ListInvoices(_ context.Context, _ billing.ListInvoicesInput) (billing.ListInvoicesResponse, error) {
	return m.invoices, nil
}

type mockChargeService struct {
	charges []charges.Charge
}

func (m *mockChargeService) ListCharges(_ context.Context, _ charges.ListChargesInput) (pagination.Result[charges.Charge], error) {
	return pagination.Result[charges.Charge]{
		Items: m.charges,
	}, nil
}

func TestLoadForSubscription_ChargeBackedGatheringLineDoesNotCollideWithCharge(t *testing.T) {
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	period := timeutil.ClosedPeriod{From: now, To: now.Add(time.Hour)}
	namespace := "test-ns"
	custID := "cust-1"
	subID := "sub-1"
	uniqueRefID := "shared-unique-ref-id"
	chargeID := "charge-1"
	gatheringInvoiceID := "gathering-inv-1"

	sub := subscription.Subscription{
		NamespacedID: models.NamespacedID{
			Namespace: namespace,
			ID:        subID,
		},
		CustomerId: custID,
	}

	// Charge-backed gathering line mirror sharing the same ChildUniqueReferenceID as the charge
	gatheringLine := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: namespace},
				ID:              "line-gathering-1",
				Name:            "Gathering Line",
			},
			InvoiceID:              gatheringInvoiceID,
			ChildUniqueReferenceID: lo.ToPtr(uniqueRefID),
			ChargeID:               lo.ToPtr(chargeID),
			ServicePeriod:          period,
		},
	}

	gatheringInvoice := billing.GatheringInvoice{
		GatheringInvoiceBase: billing.GatheringInvoiceBase{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: namespace},
				ID:              gatheringInvoiceID,
				Name:            "Gathering Invoice",
			},
			CustomerID:    custID,
			ServicePeriod: period,
		},
	}

	flatFeeIntent := flatfee.Intent{
		Intent: meta.Intent{
			ManagedBy:  billing.SubscriptionManagedLine,
			CustomerID: custID,
			Currency:   currenciestestutils.NewFiatCurrency(t, "USD"),
			TaxConfig: productcatalog.TaxCodeConfig{
				TaxCodeID: "tax-code-1",
			},
			UniqueReferenceID: lo.ToPtr(uniqueRefID),
		},
		IntentMutableFields: flatfee.IntentMutableFields{
			IntentMutableFields: meta.IntentMutableFields{
				Name:              "flat fee",
				ServicePeriod:     period,
				FullServicePeriod: period,
				BillingPeriod:     period,
			},
			InvoiceAt:             period.From,
			PaymentTerm:           productcatalog.InAdvancePaymentTerm,
			AmountBeforeProration: alpacadecimal.NewFromInt(100),
		},
		SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
	}

	flatFeeCharge := flatfee.Charge{
		ChargeBase: flatfee.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: namespace},
				ManagedModel: models.ManagedModel{
					CreatedAt: now,
					UpdatedAt: now,
				},
				ID: chargeID,
			},
			Intent: flatfee.NewOverridableIntent(flatFeeIntent, nil),
			Status: flatfee.StatusActive,
		},
	}

	billingSvc := &mockBillingService{
		gatheringLines: billing.GatheringLines{gatheringLine},
		invoices: billing.ListInvoicesResponse{
			Items: []billing.Invoice{billing.NewInvoice(gatheringInvoice)},
		},
	}
	chargeSvc := &mockChargeService{
		charges: []charges.Charge{charges.NewCharge(flatFeeCharge)},
	}

	loader := NewLoader(billingSvc, chargeSvc)
	state, err := loader.LoadForSubscription(context.Background(), sub)
	require.NoError(t, err)

	// Verify that the charge item is indexed and no collision occurred
	require.Contains(t, state.ByUniqueID, uniqueRefID)
	item := state.ByUniqueID[uniqueRefID]
	require.Equal(t, ItemTypeChargeFlatFee, item.Type())
	require.Equal(t, chargeID, item.ID().ID)

	// Verify that the gathering invoice is loaded in the state
	require.Contains(t, state.Invoices, gatheringInvoiceID)
}
