package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
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
	require.Equal(t, 1, lineageService.backfillCalls)
	require.Equal(t, charge.Namespace, lineageService.backfillInput.Namespace)
	require.Equal(t, charge.Intent.CustomerID, lineageService.backfillInput.CustomerID)
	require.Equal(t, charge.Intent.Currency, lineageService.backfillInput.Currency)
	require.True(t, charge.Intent.CreditAmount.Equal(lineageService.backfillInput.Amount))
	require.Equal(t, "ledger-tx-1", lineageService.backfillInput.BackingTransactionGroupID)
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
	updateChargeCalls      int
	updatedBase            creditpurchase.ChargeBase
	createCreditGrantCalls int
	createdGrantChargeID   meta.ChargeID
	createdGrantInput      creditpurchase.CreateCreditGrantInput
}

func (a *promotionalStateMachineAdapter) CreateCharge(ctx context.Context, in creditpurchase.CreateChargeInput) (creditpurchase.Charge, error) {
	return creditpurchase.Charge{}, errors.New("unexpected CreateCharge call")
}

func (a *promotionalStateMachineAdapter) UpdateCharge(ctx context.Context, charge creditpurchase.ChargeBase) (creditpurchase.ChargeBase, error) {
	a.updateChargeCalls++
	a.updatedBase = charge
	return charge, nil
}

func (a *promotionalStateMachineAdapter) GetByIDs(ctx context.Context, ids creditpurchase.GetByIDsInput) ([]creditpurchase.Charge, error) {
	return nil, errors.New("unexpected GetByIDs call")
}

func (a *promotionalStateMachineAdapter) GetByID(ctx context.Context, id creditpurchase.GetByIDInput) (creditpurchase.Charge, error) {
	return creditpurchase.Charge{}, errors.New("unexpected GetByID call")
}

func (a *promotionalStateMachineAdapter) ListCharges(ctx context.Context, input creditpurchase.ListChargesInput) (pagination.Result[creditpurchase.Charge], error) {
	return pagination.Result[creditpurchase.Charge]{}, errors.New("unexpected ListCharges call")
}

func (a *promotionalStateMachineAdapter) ListFundedCreditActivities(ctx context.Context, input creditpurchase.ListFundedCreditActivitiesInput) (creditpurchase.ListFundedCreditActivitiesResult, error) {
	return creditpurchase.ListFundedCreditActivitiesResult{}, errors.New("unexpected ListFundedCreditActivities call")
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

func (a *promotionalStateMachineAdapter) CreateExternalPayment(ctx context.Context, chargeID meta.ChargeID, input payment.ExternalCreateInput) (payment.External, error) {
	return payment.External{}, errors.New("unexpected CreateExternalPayment call")
}

func (a *promotionalStateMachineAdapter) UpdateExternalPayment(ctx context.Context, input payment.External) (payment.External, error) {
	return input, errors.New("unexpected UpdateExternalPayment call")
}

func (a *promotionalStateMachineAdapter) CreateInvoicedPayment(ctx context.Context, chargeID meta.ChargeID, input payment.InvoicedCreate) (payment.Invoiced, error) {
	return payment.Invoiced{}, errors.New("unexpected CreateInvoicedPayment call")
}

func (a *promotionalStateMachineAdapter) UpdateInvoicedPayment(ctx context.Context, input payment.Invoiced) (payment.Invoiced, error) {
	return input, errors.New("unexpected UpdateInvoicedPayment call")
}

func (a *promotionalStateMachineAdapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	return nil, nil, errors.New("unexpected Tx call")
}

type promotionalStateMachineHandler struct {
	transactionGroupID string
	promotionalCalls   int
	promotionalCharge  creditpurchase.Charge
}

func (h *promotionalStateMachineHandler) OnPromotionalCreditPurchase(ctx context.Context, charge creditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	h.promotionalCalls++
	h.promotionalCharge = charge
	return ledgertransaction.GroupReference{TransactionGroupID: h.transactionGroupID}, nil
}

func (h *promotionalStateMachineHandler) OnCreditPurchaseInitiated(ctx context.Context, charge creditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	return ledgertransaction.GroupReference{}, errors.New("unexpected OnCreditPurchaseInitiated call")
}

func (h *promotionalStateMachineHandler) OnCreditPurchasePaymentAuthorized(ctx context.Context, input creditpurchase.PaymentEventInput) (ledgertransaction.GroupReference, error) {
	return ledgertransaction.GroupReference{}, errors.New("unexpected OnCreditPurchasePaymentAuthorized call")
}

func (h *promotionalStateMachineHandler) OnCreditPurchasePaymentSettled(ctx context.Context, input creditpurchase.PaymentEventInput) (ledgertransaction.GroupReference, error) {
	return ledgertransaction.GroupReference{}, errors.New("unexpected OnCreditPurchasePaymentSettled call")
}

type promotionalStateMachineLineage struct {
	backfillCalls int
	backfillInput lineage.BackfillAdvanceLineageSegmentsInput
}

func (l *promotionalStateMachineLineage) CreateInitialLineages(ctx context.Context, input lineage.CreateInitialLineagesInput) error {
	return errors.New("unexpected CreateInitialLineages call")
}

func (l *promotionalStateMachineLineage) LoadActiveSegmentsByRealizationID(ctx context.Context, namespace string, realizationIDs []string) (lineage.ActiveSegmentsByRealizationID, error) {
	return nil, errors.New("unexpected LoadActiveSegmentsByRealizationID call")
}

func (l *promotionalStateMachineLineage) LoadLineagesByCustomer(ctx context.Context, input lineage.LoadLineagesByCustomerInput) ([]lineage.Lineage, error) {
	return nil, errors.New("unexpected LoadLineagesByCustomer call")
}

func (l *promotionalStateMachineLineage) PersistCorrectionLineageSegments(ctx context.Context, input lineage.PersistCorrectionLineageSegmentsInput) error {
	return errors.New("unexpected PersistCorrectionLineageSegments call")
}

func (l *promotionalStateMachineLineage) BackfillAdvanceLineageSegments(ctx context.Context, input lineage.BackfillAdvanceLineageSegmentsInput) error {
	l.backfillCalls++
	l.backfillInput = input
	return nil
}

func (l *promotionalStateMachineLineage) CloseSegment(ctx context.Context, segmentID string, closedAt time.Time) error {
	return errors.New("unexpected CloseSegment call")
}

func (l *promotionalStateMachineLineage) CreateSegment(ctx context.Context, input lineage.CreateSegmentInput) error {
	return errors.New("unexpected CreateSegment call")
}

var _ creditpurchase.Adapter = (*promotionalStateMachineAdapter)(nil)
var _ creditpurchase.Handler = (*promotionalStateMachineHandler)(nil)
var _ lineage.Service = (*promotionalStateMachineLineage)(nil)
