package service

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestPromotionalCreditPurchaseStateMachineAdvancesCreatedChargeToFinal(t *testing.T) {
	// given:
	// - a created promotional credit-purchase charge
	// when:
	// - the promotional state machine advances until stable
	// then:
	// - it grants the promotional credits, backfills lineage, and persists the final status
	stateMachine, charge, adapter, lineageService := newPromotionalStateMachineTestMachine(
		t,
		creditpurchase.StatusCreated,
	)

	advancedCharge, err := stateMachine.AdvanceUntilStateStable(t.Context())

	require.NoError(t, err)
	require.NotNil(t, advancedCharge)
	require.Equal(t, creditpurchase.StatusFinal, advancedCharge.Status)
	require.Equal(t, creditpurchase.StatusFinal, adapter.updatedBase.Status)
	require.NotNil(t, advancedCharge.Realizations.CreditGrantRealization)
	require.NotEmpty(t, advancedCharge.Realizations.CreditGrantRealization.TransactionGroupID)
	require.Equal(t, 1, adapter.createCreditGrantCalls)
	require.Equal(t, charge.GetChargeID(), adapter.createdGrantChargeID)
	require.Equal(t, advancedCharge.Realizations.CreditGrantRealization.TransactionGroupID, adapter.createdGrantInput.TransactionGroupID)
	require.False(t, adapter.createdGrantInput.GrantedAt.IsZero())
	lineageService.AssertExpectations(t)
}

func TestPromotionalCreditPurchaseStateMachineAdvancesActiveChargeToFinal(t *testing.T) {
	// given:
	// - an active promotional credit-purchase charge
	// when:
	// - the promotional state machine advances until stable
	// then:
	// - it still grants once and persists the final status
	stateMachine, _, adapter, lineageService := newPromotionalStateMachineTestMachine(
		t,
		creditpurchase.StatusActive,
	)

	advancedCharge, err := stateMachine.AdvanceUntilStateStable(t.Context())

	require.NoError(t, err)
	require.NotNil(t, advancedCharge)
	require.Equal(t, creditpurchase.StatusFinal, advancedCharge.Status)
	require.Equal(t, 1, adapter.createCreditGrantCalls)
	require.Equal(t, 1, adapter.updateChargeCalls)
	lineageService.AssertExpectations(t)
}

func TestPromotionalCreditPurchaseStateMachineRejectsExistingCreditGrant(t *testing.T) {
	// given:
	// - a promotional charge that already has a credit grant realization
	// when:
	// - the promotional state machine attempts to grant credits
	// then:
	// - it fails before creating another grant
	charge := newPromotionalStateMachineTestCharge(creditpurchase.StatusCreated)
	charge.Realizations.CreditGrantRealization = &ledgertransaction.TimedGroupReference{
		GroupReference: ledgertransaction.GroupReference{
			TransactionGroupID: "existing-ledger-tx",
		},
		Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	adapter := &promotionalStateMachineAdapter{}
	lineageService := &promotionalStateMachineLineage{}
	svc := &service{
		adapter: adapter,
		handler: &promotionalStateMachineHandler{},
		lineage: lineageService,
	}

	stateMachine, err := NewPromotionalCreditPurchaseStateMachine(StateMachineConfig{
		Charge:  charge,
		Adapter: adapter,
		Service: svc,
	})
	require.NoError(t, err)

	advancedCharge, err := stateMachine.AdvanceUntilStateStable(t.Context())

	require.Error(t, err)
	require.ErrorContains(t, err, "promotional credit grant already realized")
	require.Nil(t, advancedCharge)
	require.Zero(t, adapter.createCreditGrantCalls)
	require.Zero(t, adapter.updateChargeCalls)
	lineageService.AssertNotCalled(t, "BackfillAdvanceLineageSegments", mock.Anything, mock.Anything)
}

func TestPromotionalCreditPurchaseStateMachineReturnsNilForFinalCharge(t *testing.T) {
	// given:
	// - a final promotional credit-purchase charge
	// when:
	// - the promotional state machine advances until stable
	// then:
	// - it is already stable and does not call side-effect handlers
	adapter := &promotionalStateMachineAdapter{}
	svc := &service{
		adapter: adapter,
		handler: &promotionalStateMachineHandler{},
		lineage: &promotionalStateMachineLineage{},
	}

	stateMachine, err := NewPromotionalCreditPurchaseStateMachine(StateMachineConfig{
		Charge:  newPromotionalStateMachineTestCharge(creditpurchase.StatusFinal),
		Adapter: adapter,
		Service: svc,
	})
	require.NoError(t, err)

	advancedCharge, err := stateMachine.AdvanceUntilStateStable(t.Context())

	require.NoError(t, err)
	require.Nil(t, advancedCharge)
	require.Zero(t, adapter.createCreditGrantCalls)
	require.Zero(t, adapter.updateChargeCalls)
}

func TestPromotionalCreditPurchaseStateMachineRejectsNonPromotionalCharge(t *testing.T) {
	// given:
	// - a credit-purchase charge with invoice settlement
	// when:
	// - the promotional state machine is constructed
	// then:
	// - construction fails before any lifecycle side effect can happen
	charge := newPromotionalStateMachineTestCharge(creditpurchase.StatusCreated)
	charge.Intent.Settlement = creditpurchase.NewSettlement(creditpurchase.InvoiceSettlement{})

	_, err := NewPromotionalCreditPurchaseStateMachine(StateMachineConfig{
		Charge:  charge,
		Adapter: &promotionalStateMachineAdapter{},
		Service: &service{},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "is not promotional")
}

func TestPromotionalCreditPurchaseStateMachineRejectsMissingAdapter(t *testing.T) {
	// given:
	// - a promotional credit-purchase charge without persistence
	// when:
	// - the promotional state machine is constructed
	// then:
	// - construction fails before lifecycle methods can dereference the adapter
	_, err := NewPromotionalCreditPurchaseStateMachine(StateMachineConfig{
		Charge:  newPromotionalStateMachineTestCharge(creditpurchase.StatusCreated),
		Service: &service{},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "adapter is required")
}

func TestPromotionalCreditPurchaseStateMachineRejectsMissingService(t *testing.T) {
	// given:
	// - a promotional credit-purchase charge without runtime service dependencies
	// when:
	// - the promotional state machine is constructed
	// then:
	// - construction fails before final-state entry can dereference the service
	_, err := NewPromotionalCreditPurchaseStateMachine(StateMachineConfig{
		Charge:  newPromotionalStateMachineTestCharge(creditpurchase.StatusCreated),
		Adapter: &promotionalStateMachineAdapter{},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "service is required")
}

func newPromotionalStateMachineTestMachine(
	t *testing.T,
	status creditpurchase.Status,
) (*PromotionalCreditpurchaseStateMachine, creditpurchase.Charge, *promotionalStateMachineAdapter, *promotionalStateMachineLineage) {
	t.Helper()

	charge := newPromotionalStateMachineTestCharge(status)
	adapter := &promotionalStateMachineAdapter{}
	lineageService := &promotionalStateMachineLineage{}
	handler := &promotionalStateMachineHandler{
		onPromotionalCreditPurchase: func(ctx context.Context, charge creditpurchase.Charge) (ledgertransaction.GroupReference, error) {
			return ledgertransaction.GroupReference{
				TransactionGroupID: "ledger-tx-1",
			}, nil
		},
	}
	svc := &service{
		adapter: adapter,
		handler: handler,
		lineage: lineageService,
	}

	lineageService.On("BackfillAdvanceLineageSegments",
		mock.Anything,
		mock.MatchedBy(func(input lineage.BackfillAdvanceLineageSegmentsInput) bool {
			return input.Namespace == charge.Namespace &&
				input.CustomerID == charge.Intent.CustomerID &&
				input.Currency == charge.Intent.Currency &&
				input.Amount.Equal(charge.Intent.CreditAmount) &&
				input.BackingTransactionGroupID != ""
		})).
		Return(nil).
		Once()

	stateMachine, err := NewPromotionalCreditPurchaseStateMachine(StateMachineConfig{
		Charge:  charge,
		Adapter: adapter,
		Service: svc,
	})
	require.NoError(t, err)

	return stateMachine, charge, adapter, lineageService
}

func newPromotionalStateMachineTestCharge(status creditpurchase.Status) creditpurchase.Charge {
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	return creditpurchase.Charge{
		ChargeBase: creditpurchase.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{
					Namespace: "test-namespace",
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: period.From,
					UpdatedAt: period.From,
				},
				ID: "charge-1",
			},
			Intent: creditpurchase.Intent{
				Intent: meta.Intent{
					CustomerID: "customer-1",
					Currency:   currencyx.Code("USD"),
				},
				IntentMutableFields: meta.IntentMutableFields{
					Name:              "test promotional credits",
					ServicePeriod:     period,
					FullServicePeriod: period,
					BillingPeriod:     period,
				},
				CreditAmount: alpacadecimal.NewFromFloat(100),
				Settlement:   creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
			},
			Status: status,
		},
	}
}

type promotionalStateMachineAdapter struct {
	creditpurchase.Adapter

	updateChargeCalls      int
	updatedBase            creditpurchase.ChargeBase
	createCreditGrantCalls int
	createdGrantChargeID   meta.ChargeID
	createdGrantInput      creditpurchase.CreateCreditGrantInput
}

func (a *promotionalStateMachineAdapter) UpdateCharge(ctx context.Context, charge creditpurchase.ChargeBase) (creditpurchase.ChargeBase, error) {
	a.updateChargeCalls++
	a.updatedBase = charge
	return charge, nil
}

func (a *promotionalStateMachineAdapter) CreateCreditGrant(ctx context.Context, chargeID meta.ChargeID, input creditpurchase.CreateCreditGrantInput) (ledgertransaction.TimedGroupReference, error) {
	a.createCreditGrantCalls++
	a.createdGrantChargeID = chargeID
	a.createdGrantInput = input
	return ledgertransaction.TimedGroupReference{
		GroupReference: ledgertransaction.GroupReference{
			TransactionGroupID: input.TransactionGroupID,
		},
		Time: input.GrantedAt,
	}, nil
}

type promotionalStateMachineHandler struct {
	creditpurchase.Handler

	onPromotionalCreditPurchase func(ctx context.Context, charge creditpurchase.Charge) (ledgertransaction.GroupReference, error)
}

func (h *promotionalStateMachineHandler) OnPromotionalCreditPurchase(ctx context.Context, charge creditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	if h.onPromotionalCreditPurchase == nil {
		return ledgertransaction.GroupReference{}, nil
	}

	return h.onPromotionalCreditPurchase(ctx, charge)
}

type promotionalStateMachineLineage struct {
	lineage.Service
	mock.Mock
}

func (l *promotionalStateMachineLineage) BackfillAdvanceLineageSegments(ctx context.Context, input lineage.BackfillAdvanceLineageSegmentsInput) error {
	args := l.Called(ctx, input)
	return args.Error(0)
}

var (
	_ creditpurchase.Handler = (*promotionalStateMachineHandler)(nil)
	_ lineage.Service        = (*promotionalStateMachineLineage)(nil)
)
