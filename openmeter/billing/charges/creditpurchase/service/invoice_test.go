package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestInvoiceCreditPurchaseStateMachineGrantsCreditsWhenInvoiceIsCreated(t *testing.T) {
	// given:
	// - a newly-created invoice-settled credit purchase
	// when:
	// - billing reports that its standard invoice was created
	// then:
	// - the machine grants credits before persisting payment-pending
	charge := newInvoiceStateMachineTestCharge(t, creditpurchase.StatusCreated)
	lineWithHeader := newInvoiceStateMachineTestLine(t, charge, alpacadecimal.NewFromInt(50))
	adapter := &externalStateMachineAdapter{}
	handler := &externalStateMachineHandler{}
	lineageService := &externalStateMachineLineage{}
	handler.On("OnCreditPurchaseInitiated", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			inputCharge := args.Get(1).(creditpurchase.Charge)
			require.Equal(t, creditpurchase.StatusActiveInitialCreditGrant, inputCharge.Status)
			require.Nil(t, inputCharge.Realizations.CreditGrantRealization)
			require.Nil(t, inputCharge.Realizations.InvoiceSettlement)
		}).
		Return(ledgertransaction.GroupReference{TransactionGroupID: "initiated-ledger-tx"}, nil).
		Once()
	lineageService.On("BackfillAdvanceLineageSegments", mock.Anything, mock.Anything).
		Return(nil).
		Once()

	service := newInvoiceStateMachineTestService(t, adapter, handler, lineageService)

	updatedCharge, err := service.handleInvoiceLifecycleTrigger(t.Context(), HandleInvoiceLifecycleTriggerInput{
		Charge:         charge,
		Trigger:        meta.TriggerInvoiceCreated,
		LineWithHeader: lineWithHeader,
	})

	require.NoError(t, err)
	require.Equal(t, creditpurchase.StatusActivePaymentPending, updatedCharge.Status)
	require.NotNil(t, updatedCharge.Realizations.CreditGrantRealization)
	require.Equal(t, "initiated-ledger-tx", updatedCharge.Realizations.CreditGrantRealization.TransactionGroupID)
	require.Equal(t, []creditpurchase.Status{
		creditpurchase.StatusActiveInitialCreditGrant,
		creditpurchase.StatusActivePaymentPending,
	}, adapter.updatedBaseStatuses)
	require.Equal(t, 1, adapter.createCreditGrantCalls)
	handler.AssertExpectations(t)
	lineageService.AssertExpectations(t)
}

func TestInvoiceCreditPurchaseStateMachineAuthorizesAndSettlesPayment(t *testing.T) {
	// given:
	// - an invoice-settled credit purchase with granted credits
	// when:
	// - billing authorizes and then settles its invoice payment
	// then:
	// - the exact invoice amount is persisted and the charge becomes final
	charge := newGrantedInvoiceStateMachineTestCharge(t, creditpurchase.StatusActivePaymentPending)
	fiatAmount := alpacadecimal.RequireFromString("49.99")
	lineWithHeader := newInvoiceStateMachineTestLine(t, charge, fiatAmount)
	adapter := &externalStateMachineAdapter{}
	handler := &externalStateMachineHandler{}
	lineageService := &externalStateMachineLineage{}
	handler.On("OnCreditPurchasePaymentAuthorized", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			input := args.Get(1).(creditpurchase.PaymentEventInput)
			require.Equal(t, creditpurchase.StatusActivePaymentAuthorized, input.Charge.Status)
			require.NotNil(t, input.Charge.Realizations.CreditGrantRealization)
			require.Nil(t, input.Charge.Realizations.InvoiceSettlement)
			require.True(t, input.FiatAmount.Equal(fiatAmount))
		}).
		Return(ledgertransaction.GroupReference{TransactionGroupID: "authorized-ledger-tx"}, nil).
		Once()
	handler.On("OnCreditPurchasePaymentSettled", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			input := args.Get(1).(creditpurchase.PaymentEventInput)
			require.Equal(t, creditpurchase.StatusActivePaymentSettled, input.Charge.Status)
			require.NotNil(t, input.Charge.Realizations.InvoiceSettlement)
			require.Equal(t, payment.StatusAuthorized, input.Charge.Realizations.InvoiceSettlement.Status)
			require.True(t, input.FiatAmount.Equal(fiatAmount))
		}).
		Return(ledgertransaction.GroupReference{TransactionGroupID: "settled-ledger-tx"}, nil).
		Once()

	service := newInvoiceStateMachineTestService(t, adapter, handler, lineageService)

	authorizedCharge, err := service.handleInvoiceLifecycleTrigger(t.Context(), HandleInvoiceLifecycleTriggerInput{
		Charge:         charge,
		Trigger:        billing.TriggerAuthorized,
		LineWithHeader: lineWithHeader,
	})
	require.NoError(t, err)
	require.Equal(t, creditpurchase.StatusActivePaymentAuthorized, authorizedCharge.Status)
	require.NotNil(t, authorizedCharge.Realizations.InvoiceSettlement)
	require.Equal(t, payment.StatusAuthorized, authorizedCharge.Realizations.InvoiceSettlement.Status)
	require.True(t, authorizedCharge.Realizations.InvoiceSettlement.FiatAmount.Equal(fiatAmount))
	require.Equal(t, lineWithHeader.Line.ID, authorizedCharge.Realizations.InvoiceSettlement.LineID)
	require.Equal(t, lineWithHeader.Invoice.ID, authorizedCharge.Realizations.InvoiceSettlement.InvoiceID)

	settledCharge, err := service.handleInvoiceLifecycleTrigger(t.Context(), HandleInvoiceLifecycleTriggerInput{
		Charge:         authorizedCharge,
		Trigger:        billing.TriggerPaid,
		LineWithHeader: lineWithHeader,
	})
	require.NoError(t, err)
	require.Equal(t, creditpurchase.StatusFinal, settledCharge.Status)
	require.NotNil(t, settledCharge.Realizations.InvoiceSettlement)
	require.Equal(t, payment.StatusSettled, settledCharge.Realizations.InvoiceSettlement.Status)
	require.Equal(t, "settled-ledger-tx", settledCharge.Realizations.InvoiceSettlement.Settled.TransactionGroupID)
	require.Equal(t, 1, adapter.createInvoicedPaymentCalls)
	require.Equal(t, 1, adapter.updateInvoicedPaymentCalls)
	require.Equal(t, []creditpurchase.Status{
		creditpurchase.StatusActivePaymentAuthorized,
		creditpurchase.StatusActivePaymentSettled,
		creditpurchase.StatusFinal,
	}, adapter.updatedBaseStatuses)
	handler.AssertExpectations(t)
}

func TestInvoiceCreditPurchasePaymentPassesExactFiatAmount(t *testing.T) {
	charge := newGrantedInvoiceStateMachineTestCharge(t, creditpurchase.StatusActivePaymentPending)
	fiatAmount := alpacadecimal.RequireFromString("49.99")
	lineWithHeader := newInvoiceStateMachineTestLine(t, charge, fiatAmount)
	handler := &externalStateMachineHandler{}
	handler.On("OnCreditPurchasePaymentAuthorized", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			input := args.Get(1).(creditpurchase.PaymentEventInput)
			require.Equal(t, creditpurchase.StatusActivePaymentAuthorized, input.Charge.Status)
			require.True(t, input.FiatAmount.Equal(fiatAmount))
		}).
		Return(ledgertransaction.GroupReference{TransactionGroupID: "authorized-ledger-tx"}, nil).
		Once()
	handler.On("OnCreditPurchasePaymentSettled", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			input := args.Get(1).(creditpurchase.PaymentEventInput)
			require.Equal(t, creditpurchase.StatusActivePaymentSettled, input.Charge.Status)
			require.True(t, input.FiatAmount.Equal(fiatAmount))
		}).
		Return(ledgertransaction.GroupReference{TransactionGroupID: "settled-ledger-tx"}, nil).
		Once()
	adapter := &externalStateMachineAdapter{}
	realizationsService := newExternalStateMachineRealizations(t, adapter, handler, &externalStateMachineLineage{})
	svc := &service{
		adapter:      adapter,
		handler:      handler,
		realizations: realizationsService,
	}

	err := svc.PostInvoicePaymentAuthorized(t.Context(), charge, lineWithHeader)
	require.NoError(t, err)
	require.True(t, adapter.createdInvoicedPayment.FiatAmount.Equal(fiatAmount))

	charge.Status = creditpurchase.StatusActivePaymentAuthorized
	charge.Realizations.InvoiceSettlement = &adapter.createdInvoicedPayment
	err = svc.PostInvoicePaymentSettled(t.Context(), charge, lineWithHeader)
	require.NoError(t, err)
	require.True(t, adapter.updatedInvoicedPayment.FiatAmount.Equal(fiatAmount))
	handler.AssertExpectations(t)
}

func TestInvoiceCreditPurchaseStateMachineMapsLegacyActiveStatusAtConstruction(t *testing.T) {
	t.Run("payment pending", func(t *testing.T) {
		// given:
		// - a historical active invoice credit purchase with a grant and no payment
		// when:
		// - billing authorizes payment
		// then:
		// - the machine maps payment-pending in memory and persists authorization directly
		charge := newGrantedInvoiceStateMachineTestCharge(t, creditpurchase.StatusActive)
		lineWithHeader := newInvoiceStateMachineTestLine(t, charge, alpacadecimal.NewFromInt(50))
		adapter := &externalStateMachineAdapter{}
		handler := &externalStateMachineHandler{}
		handler.On("OnCreditPurchasePaymentAuthorized", mock.Anything, mock.Anything).
			Return(ledgertransaction.GroupReference{TransactionGroupID: "authorized-ledger-tx"}, nil).
			Once()

		service := newInvoiceStateMachineTestService(t, adapter, handler, &externalStateMachineLineage{})

		updatedCharge, err := service.handleInvoiceLifecycleTrigger(t.Context(), HandleInvoiceLifecycleTriggerInput{
			Charge:         charge,
			Trigger:        billing.TriggerAuthorized,
			LineWithHeader: lineWithHeader,
		})

		require.NoError(t, err)
		require.Equal(t, creditpurchase.StatusActivePaymentAuthorized, updatedCharge.Status)
		require.Zero(t, adapter.createCreditGrantCalls)
		require.Equal(t, []creditpurchase.Status{
			creditpurchase.StatusActivePaymentAuthorized,
		}, adapter.updatedBaseStatuses)
		handler.AssertExpectations(t)
	})

	t.Run("payment authorized", func(t *testing.T) {
		// given:
		// - a historical active invoice credit purchase with an authorized payment
		// when:
		// - billing settles payment
		// then:
		// - the machine maps payment-authorized in memory and settles without reauthorizing
		charge := newGrantedInvoiceStateMachineTestCharge(t, creditpurchase.StatusActive)
		lineWithHeader := newInvoiceStateMachineTestLine(t, charge, alpacadecimal.NewFromInt(50))
		charge.Realizations.InvoiceSettlement = newAuthorizedInvoiceStateMachinePayment(charge, lineWithHeader)
		adapter := &externalStateMachineAdapter{}
		handler := &externalStateMachineHandler{}
		handler.On("OnCreditPurchasePaymentSettled", mock.Anything, mock.Anything).
			Return(ledgertransaction.GroupReference{TransactionGroupID: "settled-ledger-tx"}, nil).
			Once()

		service := newInvoiceStateMachineTestService(t, adapter, handler, &externalStateMachineLineage{})

		updatedCharge, err := service.handleInvoiceLifecycleTrigger(t.Context(), HandleInvoiceLifecycleTriggerInput{
			Charge:         charge,
			Trigger:        billing.TriggerPaid,
			LineWithHeader: lineWithHeader,
		})

		require.NoError(t, err)
		require.Equal(t, creditpurchase.StatusFinal, updatedCharge.Status)
		require.Zero(t, adapter.createInvoicedPaymentCalls)
		require.Equal(t, 1, adapter.updateInvoicedPaymentCalls)
		require.Equal(t, []creditpurchase.Status{
			creditpurchase.StatusActivePaymentSettled,
			creditpurchase.StatusFinal,
		}, adapter.updatedBaseStatuses)
		handler.AssertNotCalled(t, "OnCreditPurchasePaymentAuthorized", mock.Anything, mock.Anything)
		handler.AssertExpectations(t)
	})
}

func TestInvoiceCreditPurchaseStateMachineRejectsLegacyActiveStatusWithoutGrant(t *testing.T) {
	// given:
	// - an inconsistent historical active invoice credit purchase without a grant
	// when:
	// - the machine attempts to reconcile it before authorization
	// then:
	// - it reports the missing durable fact instead of creating new credit
	charge := newInvoiceStateMachineTestCharge(t, creditpurchase.StatusActive)
	lineWithHeader := newInvoiceStateMachineTestLine(t, charge, alpacadecimal.NewFromInt(50))
	adapter := &externalStateMachineAdapter{}
	handler := &externalStateMachineHandler{}

	service := newInvoiceStateMachineTestService(t, adapter, handler, &externalStateMachineLineage{})

	_, err := service.handleInvoiceLifecycleTrigger(t.Context(), HandleInvoiceLifecycleTriggerInput{
		Charge:         charge,
		Trigger:        billing.TriggerAuthorized,
		LineWithHeader: lineWithHeader,
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "has no credit grant")
	require.Zero(t, adapter.createCreditGrantCalls)
	require.Zero(t, adapter.createInvoicedPaymentCalls)
	require.Zero(t, adapter.updateChargeCalls)
}

func TestInvoiceCreditPurchaseStateMachineRejectsMismatchedPaymentLine(t *testing.T) {
	charge := newGrantedInvoiceStateMachineTestCharge(t, creditpurchase.StatusActivePaymentAuthorized)
	lineWithHeader := newInvoiceStateMachineTestLine(t, charge, alpacadecimal.NewFromInt(50))
	charge.Realizations.InvoiceSettlement = newAuthorizedInvoiceStateMachinePayment(charge, lineWithHeader)
	lineWithHeader.Line.ID = "different-line"
	adapter := &externalStateMachineAdapter{}
	handler := &externalStateMachineHandler{}

	service := newInvoiceStateMachineTestService(t, adapter, handler, &externalStateMachineLineage{})

	_, err := service.handleInvoiceLifecycleTrigger(t.Context(), HandleInvoiceLifecycleTriggerInput{
		Charge:         charge,
		Trigger:        billing.TriggerPaid,
		LineWithHeader: lineWithHeader,
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "payment line ID must match line")
	require.Zero(t, adapter.updateInvoicedPaymentCalls)
	require.Zero(t, adapter.updateChargeCalls)
}

func newInvoiceStateMachineTestService(
	t *testing.T,
	adapter *externalStateMachineAdapter,
	handler *externalStateMachineHandler,
	lineageService *externalStateMachineLineage,
) *service {
	t.Helper()

	return &service{
		adapter:      adapter,
		realizations: newExternalStateMachineRealizations(t, adapter, handler, lineageService),
	}
}

func newInvoiceStateMachineTestCharge(t *testing.T, status creditpurchase.Status) creditpurchase.Charge {
	t.Helper()

	charge := newExternalStateMachineTestCharge(t, status, alpacadecimal.NewFromFloat(0.5))
	charge.Intent.Name = "test invoice credits"
	charge.Intent.Settlement = creditpurchase.NewSettlement(creditpurchase.InvoiceSettlement{
		GenericSettlement: creditpurchase.GenericSettlement{
			Currency:  currencyx.FiatCode("USD"),
			CostBasis: alpacadecimal.NewFromFloat(0.5),
		},
	})

	return charge
}

func newGrantedInvoiceStateMachineTestCharge(t *testing.T, status creditpurchase.Status) creditpurchase.Charge {
	t.Helper()

	charge := newInvoiceStateMachineTestCharge(t, status)
	charge.Realizations.CreditGrantRealization = &ledgertransaction.TimedGroupReference{
		GroupReference: ledgertransaction.GroupReference{TransactionGroupID: "initiated-ledger-tx"},
		Time:           time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	return charge
}

func newInvoiceStateMachineTestLine(t *testing.T, charge creditpurchase.Charge, fiatAmount alpacadecimal.Decimal) billing.StandardLineWithInvoiceHeader {
	t.Helper()

	invoiceID := "invoice-1"
	chargeID := charge.ID
	createdAt := charge.Intent.ServicePeriod.From
	line := &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: charge.Namespace},
				ManagedModel: models.ManagedModel{
					CreatedAt: createdAt,
					UpdatedAt: createdAt,
				},
				ID:   "line-1",
				Name: "invoice credit purchase",
			},
			ManagedBy: billing.SystemManagedLine,
			Engine:    billing.LineEngineTypeChargeCreditPurchase,
			InvoiceID: invoiceID,
			Currency:  currencyx.FiatCode("USD"),
			Period:    charge.Intent.ServicePeriod,
			InvoiceAt: createdAt,
			ChargeID:  &chargeID,
			Totals: totals.Totals{
				Amount: fiatAmount,
				Total:  fiatAmount,
			},
		},
		UsageBased: &billing.UsageBasedLine{
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      fiatAmount,
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Quantity: lo.ToPtr(alpacadecimal.NewFromInt(1)),
		},
	}

	lineWithHeader := billing.StandardLineWithInvoiceHeader{
		Line: line,
		Invoice: billing.StandardInvoice{
			StandardInvoiceBase: billing.StandardInvoiceBase{
				Namespace: charge.Namespace,
				ID:        invoiceID,
			},
		},
	}
	require.NoError(t, lineWithHeader.Validate())

	return lineWithHeader
}

func newAuthorizedInvoiceStateMachinePayment(charge creditpurchase.Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) *payment.Invoiced {
	return &payment.Invoiced{
		Payment: payment.Payment{
			NamespacedID: models.NamespacedID{Namespace: charge.Namespace, ID: "invoice-payment-1"},
			ManagedModel: models.ManagedModel{
				CreatedAt: charge.Intent.ServicePeriod.From,
				UpdatedAt: charge.Intent.ServicePeriod.From,
			},
			Base: payment.Base{
				ServicePeriod: charge.Intent.ServicePeriod,
				Status:        payment.StatusAuthorized,
				FiatAmount:    lineWithHeader.Line.Totals.Total,
				Authorized: &ledgertransaction.TimedGroupReference{
					GroupReference: ledgertransaction.GroupReference{TransactionGroupID: "authorized-ledger-tx"},
					Time:           charge.Intent.ServicePeriod.From,
				},
			},
		},
		LineID:    lineWithHeader.Line.ID,
		InvoiceID: lineWithHeader.Invoice.ID,
	}
}
