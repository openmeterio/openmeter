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
	charge := newPromotionalStateMachineTestCharge(creditpurchase.StatusCreated)
	adapter := &promotionalStateMachineAdapter{}
	handler := &promotionalStateMachineHandler{transactionGroupID: "ledger-tx-1"}
	lineageService := &promotionalStateMachineLineage{}
	svc := &service{
		adapter: adapter,
		handler: handler,
		lineage: lineageService,
	}

	stateMachine, err := NewPromotionalCreditPurchaseStateMachine(StateMachineConfig{
		Charge:  charge,
		Adapter: adapter,
		Service: svc,
	})
	require.NoError(t, err)

	lineageService.On("BackfillAdvanceLineageSegments",
		mock.Anything,
		lineage.BackfillAdvanceLineageSegmentsInput{
			Namespace:                 charge.Namespace,
			CustomerID:                charge.Intent.CustomerID,
			Currency:                  charge.Intent.Currency,
			Amount:                    charge.Intent.CreditAmount,
			BackingTransactionGroupID: "ledger-tx-1",
		}).
		Return(nil).Once()

	advancedCharge, err := stateMachine.AdvanceUntilStateStable(t.Context())

	require.NoError(t, err)
	require.NotNil(t, advancedCharge)
	require.Equal(t, creditpurchase.StatusFinal, advancedCharge.Status)
	require.Equal(t, creditpurchase.StatusFinal, adapter.updatedBase.Status)
	require.Equal(t, 1, handler.promotionalCalls)
	require.Equal(t, charge.ID, handler.promotionalCharge.ID)
	require.NotNil(t, advancedCharge.Realizations.CreditGrantRealization)
	require.Equal(t, "ledger-tx-1", advancedCharge.Realizations.CreditGrantRealization.TransactionGroupID)
	require.Equal(t, 1, adapter.createCreditGrantCalls)
	require.Equal(t, charge.GetChargeID(), adapter.createdGrantChargeID)
	require.Equal(t, "ledger-tx-1", adapter.createdGrantInput.TransactionGroupID)
	require.False(t, adapter.createdGrantInput.GrantedAt.IsZero())
}

func TestPromotionalCreditPurchaseStateMachineAdvancesActiveChargeToFinal(t *testing.T) {
	// given:
	// - an active promotional credit-purchase charge
	// when:
	// - the promotional state machine advances until stable
	// then:
	// - it still grants once and persists the final status
	charge := newPromotionalStateMachineTestCharge(creditpurchase.StatusActive)
	adapter := &promotionalStateMachineAdapter{}
	handler := &promotionalStateMachineHandler{transactionGroupID: "ledger-tx-2"}
	svc := &service{
		adapter: adapter,
		handler: handler,
		lineage: &promotionalStateMachineLineage{},
	}

	stateMachine, err := NewPromotionalCreditPurchaseStateMachine(StateMachineConfig{
		Charge:  charge,
		Adapter: adapter,
		Service: svc,
	})
	require.NoError(t, err)

	advancedCharge, err := stateMachine.AdvanceUntilStateStable(t.Context())

	require.NoError(t, err)
	require.NotNil(t, advancedCharge)
	require.Equal(t, creditpurchase.StatusFinal, advancedCharge.Status)
	require.Equal(t, 1, handler.promotionalCalls)
	require.Equal(t, 1, adapter.createCreditGrantCalls)
	require.Equal(t, 1, adapter.updateChargeCalls)
}

func TestPromotionalCreditPurchaseStateMachineReturnsNilForFinalCharge(t *testing.T) {
	// given:
	// - a final promotional credit-purchase charge
	// when:
	// - the promotional state machine advances until stable
	// then:
	// - it is already stable and does not call side-effect handlers
	adapter := &promotionalStateMachineAdapter{}
	handler := &promotionalStateMachineHandler{transactionGroupID: "ledger-tx-3"}
	svc := &service{
		adapter: adapter,
		handler: handler,
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
	require.Zero(t, handler.promotionalCalls)
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
					Name:              "test promotional credits",
					CustomerID:        "customer-1",
					Currency:          currencyx.Code("USD"),
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
	mock.Mock
}

func (h *promotionalStateMachineHandler) OnPromotionalCreditPurchase(ctx context.Context, charge creditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	args := h.Called(ctx, charge)
	return args.Get(0).(ledgertransaction.GroupReference), args.Error(1)
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
