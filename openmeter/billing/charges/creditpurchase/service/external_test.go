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
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestExternalCreditPurchaseStateMachineAdvancesThroughGrantToPaymentPending(t *testing.T) {
	for _, status := range []creditpurchase.Status{
		creditpurchase.StatusCreated,
		creditpurchase.StatusActive,
	} {
		t.Run(string(status), func(t *testing.T) {
			// given:
			// - an external credit-purchase charge in a pre-payment lifecycle status
			// when:
			// - the external state machine advances until stable
			// then:
			// - it enters the initial credit grant state, grants credits, then persists payment-pending
			charge := newExternalStateMachineTestChargeWithInput(externalStateMachineTestChargeInput{
				status:         status,
				costBasis:      alpacadecimal.NewFromFloat(0.5),
				creditAmount:   alpacadecimal.NewFromFloat(100),
				initialStatus:  creditpurchase.CreatedInitialPaymentSettlementStatus,
				featureFilters: creditpurchase.FeatureFilters{"storage", "api-calls", "storage"},
			})
			adapter := &externalStateMachineAdapter{}
			lineageService := &externalStateMachineLineage{}
			handler := &externalStateMachineHandler{}
			handler.On("OnCreditPurchaseInitiated", mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) {
					charge := args.Get(1).(creditpurchase.Charge)
					require.Equal(t, creditpurchase.StatusActiveInitialCreditGrant, charge.Status)
					require.Nil(t, charge.Realizations.CreditGrantRealization)
					require.Nil(t, charge.Realizations.ExternalPaymentSettlement)
				}).
				Return(ledgertransaction.GroupReference{TransactionGroupID: "initiated-ledger-tx"}, nil).
				Once()
			lineageService.On("BackfillAdvanceLineageSegments",
				mock.Anything,
				mock.MatchedBy(func(input lineage.BackfillAdvanceLineageSegmentsInput) bool {
					return input.Namespace == charge.Namespace &&
						input.CustomerID == charge.Intent.CustomerID &&
						input.Currency == charge.Intent.Currency &&
						input.Amount.Equal(charge.Intent.CreditAmount) &&
						input.BackingTransactionGroupID == "initiated-ledger-tx" &&
						len(input.FeatureFilters) == 2 &&
						input.FeatureFilters[0] == "api-calls" &&
						input.FeatureFilters[1] == "storage"
				})).
				Return(nil).
				Once()
			realizationsService := newExternalStateMachineRealizations(t, adapter, handler, lineageService)

			stateMachine, err := NewExternalCreditPurchaseStateMachine(StateMachineConfig{
				Charge:       charge,
				Adapter:      adapter,
				Realizations: realizationsService,
			})
			require.NoError(t, err)

			advancedCharge, err := stateMachine.AdvanceUntilStateStable(t.Context())

			require.NoError(t, err)
			require.NotNil(t, advancedCharge)
			require.Equal(t, creditpurchase.StatusActivePaymentPending, advancedCharge.Status)
			require.NotNil(t, advancedCharge.Realizations.CreditGrantRealization)
			require.Equal(t, "initiated-ledger-tx", advancedCharge.Realizations.CreditGrantRealization.TransactionGroupID)
			require.Nil(t, advancedCharge.Realizations.ExternalPaymentSettlement)
			require.Equal(t, 1, adapter.createCreditGrantCalls)
			require.Equal(t, 2, adapter.updateChargeCalls)
			require.Equal(t, []creditpurchase.Status{
				creditpurchase.StatusActiveInitialCreditGrant,
				creditpurchase.StatusActivePaymentPending,
			}, adapter.updatedBaseStatuses)
			handler.AssertExpectations(t)
			lineageService.AssertExpectations(t)
		})
	}
}

func TestExternalCreditPurchaseStateMachineUsesRoundedCreditAmount(t *testing.T) {
	// given:
	// - a payment-pending external credit-purchase charge with a sub-cent credit amount
	// when:
	// - the current external helpers grant credits and authorize payment
	// then:
	// - lineage and payment realization both use the currency-rounded credit amount
	expectedAmount := alpacadecimal.NewFromFloat(100.12)
	charge := newExternalStateMachineTestChargeWithInput(externalStateMachineTestChargeInput{
		status:        creditpurchase.StatusActivePaymentPending,
		costBasis:     alpacadecimal.NewFromFloat(0.5),
		creditAmount:  alpacadecimal.NewFromFloat(100.123),
		initialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
	})

	calc, err := charge.Intent.Currency.Calculator()
	require.NoError(t, err)
	require.True(t, calc.IsRoundedToPrecision(charge.Intent.CreditAmount))
	require.Equal(t, 100.12, charge.Intent.CreditAmount.InexactFloat64())

	adapter := &externalStateMachineAdapter{}
	lineageService := &externalStateMachineLineage{}
	handler := &externalStateMachineHandler{}
	handler.On("OnCreditPurchaseInitiated", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			charge := args.Get(1).(creditpurchase.Charge)
			require.Equal(t, expectedAmount.InexactFloat64(), charge.Intent.CreditAmount.InexactFloat64())
		}).
		Return(ledgertransaction.GroupReference{TransactionGroupID: "initiated-ledger-tx"}, nil).
		Once()
	handler.On("OnCreditPurchasePaymentAuthorized", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			input := args.Get(1).(creditpurchase.PaymentEventInput)
			require.Equal(t, expectedAmount.InexactFloat64(), input.Charge.Intent.CreditAmount.InexactFloat64())
		}).
		Return(ledgertransaction.GroupReference{TransactionGroupID: "authorized-ledger-tx"}, nil).
		Once()
	realizationsService := newExternalStateMachineRealizations(t, adapter, handler, lineageService)

	lineageService.On("BackfillAdvanceLineageSegments",
		mock.Anything,
		mock.MatchedBy(func(input lineage.BackfillAdvanceLineageSegmentsInput) bool {
			return input.Amount.Equal(expectedAmount)
		})).
		Return(nil).
		Once()

	stateMachine, err := NewExternalCreditPurchaseStateMachine(StateMachineConfig{
		Charge:       charge,
		Adapter:      adapter,
		Realizations: realizationsService,
	})
	require.NoError(t, err)

	err = stateMachine.GrantCredits(t.Context())
	require.NoError(t, err)

	err = stateMachine.FireAndActivate(t.Context(), billing.TriggerAuthorized)
	require.NoError(t, err)

	require.Equal(t, expectedAmount.InexactFloat64(), adapter.createdExternalPayment.Amount.InexactFloat64())
	handler.AssertExpectations(t)
	lineageService.AssertExpectations(t)
}

func TestExternalCreditPurchaseServiceRoutesInitialStatuses(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		initialStatus            creditpurchase.InitialPaymentSettlementStatus
		wantStatus               creditpurchase.Status
		wantPaymentStatus        *payment.Status
		wantAuthorizedCalls      int
		wantAuthorizedStatus     creditpurchase.Status
		wantSettledCalls         int
		wantSettledStatus        creditpurchase.Status
		wantCreatePaymentCalls   int
		wantUpdatePaymentCalls   int
		wantUpdateChargeStatuses []creditpurchase.Status
	}{
		{
			name:          "created",
			initialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
			wantStatus:    creditpurchase.StatusActivePaymentPending,
			wantUpdateChargeStatuses: []creditpurchase.Status{
				creditpurchase.StatusActiveInitialCreditGrant,
				creditpurchase.StatusActivePaymentPending,
			},
		},
		{
			name:                   "authorized",
			initialStatus:          creditpurchase.AuthorizedInitialPaymentSettlementStatus,
			wantStatus:             creditpurchase.StatusActivePaymentAuthorized,
			wantPaymentStatus:      lo.ToPtr(payment.StatusAuthorized),
			wantAuthorizedCalls:    1,
			wantAuthorizedStatus:   creditpurchase.StatusActivePaymentAuthorized,
			wantCreatePaymentCalls: 1,
			wantUpdateChargeStatuses: []creditpurchase.Status{
				creditpurchase.StatusActiveInitialCreditGrant,
				creditpurchase.StatusActivePaymentPending,
				creditpurchase.StatusActivePaymentAuthorized,
			},
		},
		{
			name:                   "settled",
			initialStatus:          creditpurchase.SettledInitialPaymentSettlementStatus,
			wantStatus:             creditpurchase.StatusFinal,
			wantPaymentStatus:      lo.ToPtr(payment.StatusSettled),
			wantAuthorizedCalls:    1,
			wantAuthorizedStatus:   creditpurchase.StatusActivePaymentPaidAndAuthorized,
			wantSettledCalls:       1,
			wantSettledStatus:      creditpurchase.StatusActivePaymentSettled,
			wantCreatePaymentCalls: 1,
			wantUpdatePaymentCalls: 1,
			wantUpdateChargeStatuses: []creditpurchase.Status{
				creditpurchase.StatusActiveInitialCreditGrant,
				creditpurchase.StatusActivePaymentPending,
				creditpurchase.StatusActivePaymentPaidAndAuthorized,
				creditpurchase.StatusActivePaymentSettled,
				creditpurchase.StatusFinal,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given:
			// - an external credit-purchase charge in the created state
			// when:
			// - the credit-purchase service starts the external lifecycle
			// then:
			// - it grants credits first, then routes the initial payment status through the expected transitions
			charge := newExternalStateMachineTestChargeWithInput(externalStateMachineTestChargeInput{
				status:        creditpurchase.StatusCreated,
				costBasis:     alpacadecimal.NewFromFloat(0.5),
				creditAmount:  alpacadecimal.NewFromFloat(100),
				initialStatus: tc.initialStatus,
			})

			adapter := &externalStateMachineAdapter{}
			lineageService := &externalStateMachineLineage{}

			handler := &externalStateMachineHandler{}
			handler.On("OnCreditPurchaseInitiated", mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) {
					charge := args.Get(1).(creditpurchase.Charge)
					require.Equal(t, creditpurchase.StatusActiveInitialCreditGrant, charge.Status)
					require.Nil(t, charge.Realizations.CreditGrantRealization)
					require.Nil(t, charge.Realizations.ExternalPaymentSettlement)
				}).
				Return(ledgertransaction.GroupReference{TransactionGroupID: "initiated-ledger-tx"}, nil).
				Once()
			lineageService.On("BackfillAdvanceLineageSegments",
				mock.Anything,
				mock.MatchedBy(func(input lineage.BackfillAdvanceLineageSegmentsInput) bool {
					return input.Namespace == charge.Namespace &&
						input.CustomerID == charge.Intent.CustomerID &&
						input.Currency == charge.Intent.Currency &&
						input.Amount.Equal(charge.Intent.CreditAmount) &&
						input.BackingTransactionGroupID == "initiated-ledger-tx" &&
						len(input.FeatureFilters) == 0
				})).
				Return(nil).
				Once()
			if tc.wantAuthorizedCalls > 0 {
				handler.On("OnCreditPurchasePaymentAuthorized", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						input := args.Get(1).(creditpurchase.PaymentEventInput)
						require.Equal(t, tc.wantAuthorizedStatus, input.Charge.Status)
						require.NotNil(t, input.Charge.Realizations.CreditGrantRealization)
						require.Nil(t, input.Charge.Realizations.ExternalPaymentSettlement)
					}).
					Return(ledgertransaction.GroupReference{TransactionGroupID: "authorized-ledger-tx"}, nil).
					Once()
			}
			if tc.wantSettledCalls > 0 {
				handler.On("OnCreditPurchasePaymentSettled", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						input := args.Get(1).(creditpurchase.PaymentEventInput)
						require.Equal(t, tc.wantSettledStatus, input.Charge.Status)
						require.NotNil(t, input.Charge.Realizations.CreditGrantRealization)
						require.NotNil(t, input.Charge.Realizations.ExternalPaymentSettlement)
						require.Equal(t, payment.StatusAuthorized, input.Charge.Realizations.ExternalPaymentSettlement.Status)
					}).
					Return(ledgertransaction.GroupReference{TransactionGroupID: "settled-ledger-tx"}, nil).
					Once()
			}
			realizationsService := newExternalStateMachineRealizations(t, adapter, handler, lineageService)
			svc := &service{
				adapter:      adapter,
				realizations: realizationsService,
			}

			got, err := svc.onExternalCreditPurchase(t.Context(), charge)

			require.NoError(t, err)
			require.Equal(t, tc.wantStatus, got.Status)
			require.NotNil(t, got.Realizations.CreditGrantRealization)
			require.Equal(t, "initiated-ledger-tx", got.Realizations.CreditGrantRealization.TransactionGroupID)
			if tc.wantPaymentStatus == nil {
				require.Nil(t, got.Realizations.ExternalPaymentSettlement)
			} else {
				require.NotNil(t, got.Realizations.ExternalPaymentSettlement)
				require.Equal(t, *tc.wantPaymentStatus, got.Realizations.ExternalPaymentSettlement.Status)
			}
			require.Equal(t, 1, adapter.createCreditGrantCalls)
			require.Equal(t, tc.wantCreatePaymentCalls, adapter.createExternalPaymentCalls)
			require.Equal(t, tc.wantUpdatePaymentCalls, adapter.updateExternalPaymentCalls)
			require.Equal(t, len(tc.wantUpdateChargeStatuses), adapter.updateChargeCalls)
			require.Equal(t, tc.wantUpdateChargeStatuses, adapter.updatedBaseStatuses)
			require.Equal(t, tc.wantStatus, adapter.updatedBase.Status)
			handler.AssertExpectations(t)
			lineageService.AssertExpectations(t)
		})
	}
}

func TestExternalCreditPurchaseStateMachineGrantCreditsRejectsInvalidExternalSettlement(t *testing.T) {
	for _, tc := range []struct {
		name          string
		costBasis     alpacadecimal.Decimal
		initialStatus creditpurchase.InitialPaymentSettlementStatus
		wantErr       string
	}{
		{
			name:          "zero cost basis",
			costBasis:     alpacadecimal.Zero,
			initialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
			wantErr:       "cost basis must be positive",
		},
		{
			name:          "negative cost basis",
			costBasis:     alpacadecimal.NewFromFloat(-0.5),
			initialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
			wantErr:       "cost basis must be positive",
		},
		{
			name:          "invalid initial status",
			costBasis:     alpacadecimal.NewFromFloat(0.5),
			initialStatus: creditpurchase.InitialPaymentSettlementStatus("invalid"),
			wantErr:       "initial status",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given:
			// - an external credit-purchase charge with invalid settlement input
			// when:
			// - the grant-credit action tries to create the credit grant realization
			// then:
			// - it fails with a validation error before creating realizations
			charge := newExternalStateMachineTestChargeWithInput(externalStateMachineTestChargeInput{
				status:        creditpurchase.StatusActivePaymentPending,
				costBasis:     tc.costBasis,
				creditAmount:  alpacadecimal.NewFromFloat(100),
				initialStatus: tc.initialStatus,
			})
			adapter := &externalStateMachineAdapter{}
			handler := &externalStateMachineHandler{}
			lineageService := &externalStateMachineLineage{}
			realizationsService := newExternalStateMachineRealizations(t, adapter, handler, lineageService)

			stateMachine, err := NewExternalCreditPurchaseStateMachine(StateMachineConfig{
				Charge:       charge,
				Adapter:      adapter,
				Realizations: realizationsService,
			})
			require.NoError(t, err)

			err = stateMachine.GrantCredits(t.Context())

			require.Error(t, err)
			require.ErrorContains(t, err, tc.wantErr)
			require.True(t, models.IsGenericValidationError(err))
			require.Zero(t, adapter.createCreditGrantCalls)
			require.Zero(t, adapter.createExternalPaymentCalls)
			require.Zero(t, adapter.updateChargeCalls)
			handler.AssertNotCalled(t, "OnCreditPurchaseInitiated", mock.Anything, mock.Anything)
			lineageService.AssertNotCalled(t, "BackfillAdvanceLineageSegments", mock.Anything, mock.Anything)
		})
	}
}

func TestExternalCreditPurchaseStateMachineAuthorizesAndSettlesPayment(t *testing.T) {
	// given:
	// - an active external credit-purchase charge with granted credits
	// when:
	// - the payment is authorized and then settled
	// then:
	// - payment realization moves to settled and the charge becomes final
	charge := newGrantedExternalCreditPurchaseCharge(creditpurchase.StatusActivePaymentPending)
	handler := &externalStateMachineHandler{}
	handler.On("OnCreditPurchasePaymentAuthorized", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			input := args.Get(1).(creditpurchase.PaymentEventInput)
			require.Equal(t, creditpurchase.StatusActivePaymentAuthorized, input.Charge.Status)
			require.NotNil(t, input.Charge.Realizations.CreditGrantRealization)
			require.Nil(t, input.Charge.Realizations.ExternalPaymentSettlement)
		}).
		Return(ledgertransaction.GroupReference{TransactionGroupID: "authorized-ledger-tx"}, nil).
		Once()
	handler.On("OnCreditPurchasePaymentSettled", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			input := args.Get(1).(creditpurchase.PaymentEventInput)
			require.Equal(t, creditpurchase.StatusActivePaymentSettled, input.Charge.Status)
			require.NotNil(t, input.Charge.Realizations.ExternalPaymentSettlement)
			require.Equal(t, payment.StatusAuthorized, input.Charge.Realizations.ExternalPaymentSettlement.Status)
			require.Equal(t, "authorized-ledger-tx", input.Charge.Realizations.ExternalPaymentSettlement.Authorized.TransactionGroupID)
		}).
		Return(ledgertransaction.GroupReference{TransactionGroupID: "settled-ledger-tx"}, nil).
		Once()
	adapter := &externalStateMachineAdapter{}
	lineageService := &externalStateMachineLineage{}
	realizationsService := newExternalStateMachineRealizations(t, adapter, handler, lineageService)

	stateMachine, err := NewExternalCreditPurchaseStateMachine(StateMachineConfig{
		Charge:       charge,
		Adapter:      adapter,
		Realizations: realizationsService,
	})
	require.NoError(t, err)

	err = stateMachine.FireAndActivate(t.Context(), billing.TriggerAuthorized)
	require.NoError(t, err)
	require.Equal(t, creditpurchase.StatusActivePaymentAuthorized, stateMachine.GetCharge().Status)
	require.NotNil(t, stateMachine.GetCharge().Realizations.ExternalPaymentSettlement)
	require.Equal(t, payment.StatusAuthorized, stateMachine.GetCharge().Realizations.ExternalPaymentSettlement.Status)
	require.Equal(t, "authorized-ledger-tx", stateMachine.GetCharge().Realizations.ExternalPaymentSettlement.Authorized.TransactionGroupID)
	require.Equal(t, 1, adapter.createExternalPaymentCalls)
	require.Equal(t, 1, adapter.updateChargeCalls)
	require.Equal(t, creditpurchase.StatusActivePaymentAuthorized, adapter.updatedBase.Status)

	settledCharge, err := stateMachine.handleExternalPaymentLifecycleTrigger(t.Context(), billing.TriggerPaid)

	require.NoError(t, err)
	require.Equal(t, creditpurchase.StatusFinal, settledCharge.Status)
	require.NotNil(t, settledCharge.Realizations.ExternalPaymentSettlement)
	require.Equal(t, payment.StatusSettled, settledCharge.Realizations.ExternalPaymentSettlement.Status)
	require.Equal(t, "authorized-ledger-tx", settledCharge.Realizations.ExternalPaymentSettlement.Authorized.TransactionGroupID)
	require.Equal(t, "settled-ledger-tx", settledCharge.Realizations.ExternalPaymentSettlement.Settled.TransactionGroupID)
	require.Equal(t, 1, adapter.updateExternalPaymentCalls)
	require.Equal(t, 3, adapter.updateChargeCalls)
	require.Equal(t, []creditpurchase.Status{
		creditpurchase.StatusActivePaymentAuthorized,
		creditpurchase.StatusActivePaymentSettled,
		creditpurchase.StatusFinal,
	}, adapter.updatedBaseStatuses)
	handler.AssertExpectations(t)
}

func TestExternalCreditPurchaseStateMachineAuthorizationUsesRealizationDuplicateGuard(t *testing.T) {
	// given:
	// - a payment-pending external credit-purchase charge that already has an authorized payment realization
	// when:
	// - the state machine receives another authorized trigger
	// then:
	// - the realization service reports the duplicate payment and the charge status is not persisted
	charge := newExternalStateMachineTestCharge(creditpurchase.StatusActivePaymentPending, alpacadecimal.NewFromFloat(0.5))
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
				Amount:        charge.Intent.CreditAmount,
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

func TestExternalCreditPurchaseStateMachineAuthorizesAndSettlesInSingleTransition(t *testing.T) {
	// given:
	// - a payment-pending external credit-purchase charge with no payment realization
	// when:
	// - the state machine receives the paid trigger
	// then:
	// - it books authorization before settlement and persists the final charge status
	charge := newGrantedExternalCreditPurchaseCharge(creditpurchase.StatusActivePaymentPending)
	handler := &externalStateMachineHandler{}
	handler.On("OnCreditPurchasePaymentAuthorized", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			input := args.Get(1).(creditpurchase.PaymentEventInput)
			require.Equal(t, creditpurchase.StatusActivePaymentPaidAndAuthorized, input.Charge.Status)
			require.Nil(t, input.Charge.Realizations.ExternalPaymentSettlement)
		}).
		Return(ledgertransaction.GroupReference{TransactionGroupID: "authorized-ledger-tx"}, nil).
		Once()
	handler.On("OnCreditPurchasePaymentSettled", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			input := args.Get(1).(creditpurchase.PaymentEventInput)
			require.Equal(t, creditpurchase.StatusActivePaymentSettled, input.Charge.Status)
			require.NotNil(t, input.Charge.Realizations.ExternalPaymentSettlement)
			require.Equal(t, payment.StatusAuthorized, input.Charge.Realizations.ExternalPaymentSettlement.Status)
			require.Equal(t, "authorized-ledger-tx", input.Charge.Realizations.ExternalPaymentSettlement.Authorized.TransactionGroupID)
		}).
		Return(ledgertransaction.GroupReference{TransactionGroupID: "settled-ledger-tx"}, nil).
		Once()
	adapter := &externalStateMachineAdapter{}
	lineageService := &externalStateMachineLineage{}
	realizationsService := newExternalStateMachineRealizations(t, adapter, handler, lineageService)

	stateMachine, err := NewExternalCreditPurchaseStateMachine(StateMachineConfig{
		Charge:       charge,
		Adapter:      adapter,
		Realizations: realizationsService,
	})
	require.NoError(t, err)

	settledCharge, err := stateMachine.handleExternalPaymentLifecycleTrigger(t.Context(), billing.TriggerPaid)

	require.NoError(t, err)
	require.Equal(t, creditpurchase.StatusFinal, settledCharge.Status)
	require.NotNil(t, settledCharge.Realizations.ExternalPaymentSettlement)
	require.Equal(t, payment.StatusSettled, settledCharge.Realizations.ExternalPaymentSettlement.Status)
	require.Equal(t, "authorized-ledger-tx", settledCharge.Realizations.ExternalPaymentSettlement.Authorized.TransactionGroupID)
	require.Equal(t, "settled-ledger-tx", settledCharge.Realizations.ExternalPaymentSettlement.Settled.TransactionGroupID)
	require.Equal(t, 1, adapter.createExternalPaymentCalls)
	require.Equal(t, 1, adapter.updateExternalPaymentCalls)
	require.Equal(t, 3, adapter.updateChargeCalls)
	require.Equal(t, []creditpurchase.Status{
		creditpurchase.StatusActivePaymentPaidAndAuthorized,
		creditpurchase.StatusActivePaymentSettled,
		creditpurchase.StatusFinal,
	}, adapter.updatedBaseStatuses)
	handler.AssertExpectations(t)
}

func newGrantedExternalCreditPurchaseCharge(status creditpurchase.Status) creditpurchase.Charge {
	charge := newExternalStateMachineTestCharge(status, alpacadecimal.NewFromFloat(0.5))
	charge.Realizations.CreditGrantRealization = &ledgertransaction.TimedGroupReference{
		GroupReference: ledgertransaction.GroupReference{TransactionGroupID: "initiated-ledger-tx"},
		Time:           time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	return charge
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

	featureFilters creditpurchase.FeatureFilters
}

func newExternalStateMachineTestCharge(status creditpurchase.Status, costBasis alpacadecimal.Decimal) creditpurchase.Charge {
	return newExternalStateMachineTestChargeWithInput(externalStateMachineTestChargeInput{
		status:        status,
		costBasis:     costBasis,
		creditAmount:  alpacadecimal.NewFromFloat(100),
		initialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
	})
}

func newExternalStateMachineTestChargeWithInput(input externalStateMachineTestChargeInput) creditpurchase.Charge {
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	intent := creditpurchase.Intent{
		Intent: meta.Intent{
			CustomerID: "customer-1",
			Currency:   currencyx.Code("USD"),
		},
		IntentMutableFields: meta.IntentMutableFields{
			Name:              "test external credits",
			ServicePeriod:     period,
			FullServicePeriod: period,
			BillingPeriod:     period,
		},
		CreditAmount:   input.creditAmount,
		FeatureFilters: input.featureFilters,
		Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
			GenericSettlement: creditpurchase.GenericSettlement{
				Currency:  currencyx.Code("USD"),
				CostBasis: input.costBasis,
			},
			InitialStatus: input.initialStatus,
		}),
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
	updatedBase         creditpurchase.ChargeBase
	updatedBaseStatuses []creditpurchase.Status

	createCreditGrantCalls int
	createdGrantChargeID   meta.ChargeID
	createdGrantInput      creditpurchase.CreateCreditGrantInput

	createExternalPaymentCalls int
	createdExternalPaymentID   meta.ChargeID
	createdExternalPayment     payment.ExternalCreateInput

	updateExternalPaymentCalls int
	updatedExternalPayment     payment.External
}

func (a *externalStateMachineAdapter) UpdateCharge(ctx context.Context, charge creditpurchase.ChargeBase) (creditpurchase.ChargeBase, error) {
	a.updateChargeCalls++
	a.updatedBase = charge
	a.updatedBaseStatuses = append(a.updatedBaseStatuses, charge.Status)
	return charge, nil
}

func (a *externalStateMachineAdapter) CreateCreditGrant(ctx context.Context, chargeID meta.ChargeID, input creditpurchase.CreateCreditGrantInput) (ledgertransaction.TimedGroupReference, error) {
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

func (a *externalStateMachineAdapter) CreateExternalPayment(ctx context.Context, chargeID meta.ChargeID, input payment.ExternalCreateInput) (payment.External, error) {
	a.createExternalPaymentCalls++
	a.createdExternalPaymentID = chargeID
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
	a.updateExternalPaymentCalls++
	a.updatedExternalPayment = paymentSettlement
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
