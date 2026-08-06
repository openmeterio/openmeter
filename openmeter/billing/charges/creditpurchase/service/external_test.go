package service

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	creditpurchaserealizations "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/service/realizations"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestExternalCreditPurchaseServiceRoutesAuthorizedInitialStatus(t *testing.T) {
	// given:
	// - a created external credit purchase whose payment is already authorized
	// when:
	// - the service starts its lifecycle
	// then:
	// - credits are granted before authorization and the charge remains active
	charge := newExternalStateMachineTestChargeWithInput(t, externalStateMachineTestChargeInput{
		status:        creditpurchase.StatusCreated,
		costBasis:     alpacadecimal.NewFromFloat(0.5),
		creditAmount:  alpacadecimal.NewFromFloat(100),
		initialStatus: creditpurchase.AuthorizedInitialPaymentSettlementStatus,
	})
	adapter := &externalStateMachineAdapter{}
	lineageService := &externalStateMachineLineage{}
	handler := &externalStateMachineHandler{}
	handler.On("OnCreditPurchaseInitiated", mock.Anything, mock.Anything).
		Return(ledgertransaction.GroupReference{TransactionGroupID: "initiated-ledger-tx"}, nil).
		Once()
	lineageService.On("BackfillAdvanceLineageSegments", mock.Anything, mock.Anything).
		Return(nil).
		Once()
	handler.On("OnCreditPurchasePaymentAuthorized", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			input := args.Get(1).(creditpurchase.PaymentEventInput)
			require.Equal(t, creditpurchase.StatusActivePaymentAuthorized, input.Charge.Status)
			require.NotNil(t, input.Charge.Realizations.CreditGrantRealization)
			require.Nil(t, input.Charge.Realizations.ExternalPaymentSettlement)
		}).
		Return(ledgertransaction.GroupReference{TransactionGroupID: "authorized-ledger-tx"}, nil).
		Once()
	realizationsService := newExternalStateMachineRealizations(t, adapter, handler, lineageService)
	svc := &service{adapter: adapter, realizations: realizationsService}

	got, err := svc.onExternalCreditPurchase(t.Context(), charge)

	require.NoError(t, err)
	require.Equal(t, creditpurchase.StatusActivePaymentAuthorized, got.Status)
	require.NotNil(t, got.Realizations.CreditGrantRealization)
	require.NotNil(t, got.Realizations.ExternalPaymentSettlement)
	require.Equal(t, payment.StatusAuthorized, got.Realizations.ExternalPaymentSettlement.Status)
	require.Equal(t, []creditpurchase.Status{
		creditpurchase.StatusActiveInitialCreditGrant,
		creditpurchase.StatusActivePaymentPending,
		creditpurchase.StatusActivePaymentAuthorized,
	}, adapter.updatedBaseStatuses)
	handler.AssertExpectations(t)
	lineageService.AssertExpectations(t)
}

func TestExternalCreditPurchaseStateMachineAuthorizationUsesRealizationDuplicateGuard(t *testing.T) {
	// given:
	// - a payment-pending external credit-purchase charge that already has an authorized payment realization
	// when:
	// - the state machine receives another authorized trigger
	// then:
	// - the realization service reports the duplicate payment and the charge status is not persisted
	charge := newExternalStateMachineTestCharge(t, creditpurchase.StatusActivePaymentPending, alpacadecimal.NewFromFloat(0.5))
	charge.Realizations.CreditGrantRealization = &ledgertransaction.TimedGroupReference{
		GroupReference: ledgertransaction.GroupReference{TransactionGroupID: "initiated-ledger-tx"},
		Time:           time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	charge.Realizations.ExternalPaymentSettlement = &payment.External{
		Payment: payment.Payment{
			NamespacedID: models.NamespacedID{
				Namespace: charge.Namespace,
				ID:        "external-payment-1",
			},
			Base: payment.Base{
				ServicePeriod: charge.Intent.ServicePeriod,
				FiatAmount:    charge.Intent.CreditAmount,
				Status:        payment.StatusAuthorized,
				Authorized: &ledgertransaction.TimedGroupReference{
					GroupReference: ledgertransaction.GroupReference{TransactionGroupID: "authorized-ledger-tx"},
					Time:           time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	adapter := &externalStateMachineAdapter{}
	lineageService := &externalStateMachineLineage{}
	handler := &externalStateMachineHandler{}
	realizationsService := newExternalStateMachineRealizations(t, adapter, handler, lineageService)

	stateMachine, err := NewExternalCreditPurchaseStateMachine(StateMachineConfig{
		Charge:       charge,
		Adapter:      adapter,
		Realizations: realizationsService,
	})
	require.NoError(t, err)

	err = stateMachine.FireAndActivate(t.Context(), billing.TriggerAuthorized)

	require.Error(t, err)
	require.ErrorIs(t, err, payment.ErrPaymentAlreadyAuthorized)
	require.Zero(t, adapter.createExternalPaymentCalls)
	require.Zero(t, adapter.updateChargeCalls)
	handler.AssertNotCalled(t, "OnCreditPurchasePaymentAuthorized", mock.Anything, mock.Anything)
}

func newExternalStateMachineRealizations(
	t *testing.T,
	adapter creditpurchase.Adapter,
	handler creditpurchase.Handler,
	lineageService lineage.Service,
) *creditpurchaserealizations.Service {
	t.Helper()

	realizationsService, err := creditpurchaserealizations.New(creditpurchaserealizations.Config{
		Adapter: adapter,
		Handler: handler,
		Lineage: lineageService,
	})
	require.NoError(t, err)

	return realizationsService
}

type externalStateMachineTestChargeInput struct {
	status        creditpurchase.Status
	costBasis     alpacadecimal.Decimal
	creditAmount  alpacadecimal.Decimal
	initialStatus creditpurchase.InitialPaymentSettlementStatus
}

func newExternalStateMachineTestCharge(t *testing.T, status creditpurchase.Status, costBasis alpacadecimal.Decimal) creditpurchase.Charge {
	t.Helper()

	return newExternalStateMachineTestChargeWithInput(t, externalStateMachineTestChargeInput{
		status:        status,
		costBasis:     costBasis,
		creditAmount:  alpacadecimal.NewFromFloat(100),
		initialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
	})
}

func newExternalStateMachineTestChargeWithInput(t *testing.T, input externalStateMachineTestChargeInput) creditpurchase.Charge {
	t.Helper()

	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	intent := creditpurchase.Intent{
		Intent: meta.Intent{
			CustomerID: "customer-1",
			Currency:   currenciestestutils.NewFiatCurrency(t, "USD"),
		},
		IntentMutableFields: creditpurchase.IntentMutableFields{
			IntentMutableFields: meta.IntentMutableFields{
				Name:              "test external credits",
				ServicePeriod:     period,
				FullServicePeriod: period,
				BillingPeriod:     period,
			},
			CreditAmount: input.creditAmount,
			Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
				InitialStatus: input.initialStatus,
			}),
		},
		CostBasis: lo.ToPtr(creditpurchase.NewCostBasis(creditpurchase.FiatCostBasis{
			Rate: input.costBasis,
		})),
	}.Normalized()

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
			Intent: intent,
			Status: input.status,
		},
	}
}

type externalStateMachineAdapter struct {
	creditpurchase.Adapter

	updateChargeCalls   int
	updatedBaseStatuses []creditpurchase.Status

	createExternalPaymentCalls int
	createdExternalPayment     payment.ExternalCreateInput

	updateInvoicedPaymentCalls int
}

func (a *externalStateMachineAdapter) UpdateCharge(ctx context.Context, charge creditpurchase.ChargeBase) (creditpurchase.ChargeBase, error) {
	a.updateChargeCalls++
	a.updatedBaseStatuses = append(a.updatedBaseStatuses, charge.Status)
	return charge, nil
}

func (a *externalStateMachineAdapter) CreateCreditGrant(ctx context.Context, _ meta.ChargeID, input creditpurchase.CreateCreditGrantInput) (ledgertransaction.TimedGroupReference, error) {
	return ledgertransaction.TimedGroupReference{
		GroupReference: ledgertransaction.GroupReference{
			TransactionGroupID: input.TransactionGroupID,
		},
		Time: input.GrantedAt,
	}, nil
}

func (a *externalStateMachineAdapter) CreateExternalPayment(ctx context.Context, _ meta.ChargeID, input payment.ExternalCreateInput) (payment.External, error) {
	a.createExternalPaymentCalls++
	a.createdExternalPayment = input
	return payment.External{
		Payment: payment.Payment{
			NamespacedID: models.NamespacedID{
				Namespace: input.Namespace,
				ID:        "external-payment-1",
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			Base: input.Base,
		},
	}, nil
}

func (a *externalStateMachineAdapter) UpdateExternalPayment(ctx context.Context, paymentSettlement payment.External) (payment.External, error) {
	return paymentSettlement, nil
}

func (a *externalStateMachineAdapter) CreateInvoicedPayment(_ context.Context, _ meta.ChargeID, input payment.InvoicedCreate) (payment.Invoiced, error) {
	createdInvoicedPayment := payment.Invoiced{
		Payment: payment.Payment{
			NamespacedID: models.NamespacedID{
				Namespace: input.Namespace,
				ID:        "invoice-payment-1",
			},
			Base: input.Base,
		},
		LineID:    input.LineID,
		InvoiceID: input.InvoiceID,
	}

	return createdInvoicedPayment, nil
}

func (a *externalStateMachineAdapter) UpdateInvoicedPayment(_ context.Context, paymentSettlement payment.Invoiced) (payment.Invoiced, error) {
	a.updateInvoicedPaymentCalls++
	return paymentSettlement, nil
}

type externalStateMachineHandler struct {
	creditpurchase.Handler
	mock.Mock
}

func (h *externalStateMachineHandler) OnCreditPurchaseInitiated(ctx context.Context, charge creditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	args := h.Called(ctx, charge)
	return args.Get(0).(ledgertransaction.GroupReference), args.Error(1)
}

func (h *externalStateMachineHandler) OnCreditPurchasePaymentAuthorized(ctx context.Context, input creditpurchase.PaymentEventInput) (ledgertransaction.GroupReference, error) {
	args := h.Called(ctx, input)
	return args.Get(0).(ledgertransaction.GroupReference), args.Error(1)
}

func (h *externalStateMachineHandler) OnCreditPurchasePaymentSettled(ctx context.Context, input creditpurchase.PaymentEventInput) (ledgertransaction.GroupReference, error) {
	args := h.Called(ctx, input)
	return args.Get(0).(ledgertransaction.GroupReference), args.Error(1)
}

type externalStateMachineLineage struct {
	lineage.Service
	mock.Mock
}

func (l *externalStateMachineLineage) BackfillAdvanceLineageSegments(ctx context.Context, input lineage.BackfillAdvanceLineageSegmentsInput) error {
	args := l.Called(ctx, input)
	return args.Error(0)
}

var (
	_ creditpurchase.Handler = (*externalStateMachineHandler)(nil)
	_ lineage.Service        = (*externalStateMachineLineage)(nil)
)
