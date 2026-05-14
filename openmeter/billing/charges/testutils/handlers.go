package testutils

import (
	"context"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
)

type MockHandlers struct {
	FlatFee        flatfee.Handler
	CreditPurchase creditpurchase.Handler
	UsageBased     usagebased.Handler
}

func NewMockHandlers() MockHandlers {
	return MockHandlers{
		FlatFee:        mockFlatFeeHandler{},
		CreditPurchase: mockCreditPurchaseHandler{},
		UsageBased:     mockUsageBasedHandler{},
	}
}

type mockFlatFeeHandler struct{}

var _ flatfee.Handler = (*mockFlatFeeHandler)(nil)

func (mockFlatFeeHandler) OnAssignedToInvoice(context.Context, flatfee.OnAssignedToInvoiceInput) (creditrealization.CreateAllocationInputs, error) {
	return nil, nil
}

func (mockFlatFeeHandler) OnInvoiceUsageAccrued(context.Context, flatfee.OnInvoiceUsageAccruedInput) (ledgertransaction.GroupReference, error) {
	return newMockLedgerTransactionGroupReference(), nil
}

func (mockFlatFeeHandler) OnCreditsOnlyUsageAccrued(_ context.Context, input flatfee.OnCreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
	return creditrealization.CreateAllocationInputs{
		{
			ServicePeriod:     input.Charge.Intent.ServicePeriod,
			LedgerTransaction: newMockLedgerTransactionGroupReference(),
			Amount:            input.AmountToAllocate,
		},
	}, nil
}

func (mockFlatFeeHandler) OnCreditsOnlyUsageAccruedCorrection(_ context.Context, input flatfee.CreditsOnlyUsageAccruedCorrectionInput) (creditrealization.CreateCorrectionInputs, error) {
	return lo.Map(input.Corrections, func(correction creditrealization.CorrectionRequestItem, _ int) creditrealization.CreateCorrectionInput {
		return creditrealization.CreateCorrectionInput{
			LedgerTransaction:     newMockLedgerTransactionGroupReference(),
			Amount:                correction.Amount,
			CorrectsRealizationID: correction.Allocation.ID,
		}
	}), nil
}

func (mockFlatFeeHandler) OnPaymentAuthorized(context.Context, flatfee.OnPaymentAuthorizedInput) (ledgertransaction.GroupReference, error) {
	return newMockLedgerTransactionGroupReference(), nil
}

func (mockFlatFeeHandler) OnPaymentSettled(context.Context, flatfee.OnPaymentSettledInput) (ledgertransaction.GroupReference, error) {
	return newMockLedgerTransactionGroupReference(), nil
}

func (mockFlatFeeHandler) OnPaymentUncollectible(context.Context, flatfee.Charge) (ledgertransaction.GroupReference, error) {
	return newMockLedgerTransactionGroupReference(), nil
}

type mockCreditPurchaseHandler struct{}

var _ creditpurchase.Handler = (*mockCreditPurchaseHandler)(nil)

func (mockCreditPurchaseHandler) OnPromotionalCreditPurchase(context.Context, creditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	return newMockLedgerTransactionGroupReference(), nil
}

func (mockCreditPurchaseHandler) OnCreditPurchaseInitiated(context.Context, creditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	return newMockLedgerTransactionGroupReference(), nil
}

func (mockCreditPurchaseHandler) OnCreditPurchasePaymentAuthorized(context.Context, creditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	return newMockLedgerTransactionGroupReference(), nil
}

func (mockCreditPurchaseHandler) OnCreditPurchasePaymentSettled(context.Context, creditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	return newMockLedgerTransactionGroupReference(), nil
}

type mockUsageBasedHandler struct{}

var _ usagebased.Handler = (*mockUsageBasedHandler)(nil)

func (mockUsageBasedHandler) OnInvoiceUsageAccrued(context.Context, usagebased.OnInvoiceUsageAccruedInput) (ledgertransaction.GroupReference, error) {
	return newMockLedgerTransactionGroupReference(), nil
}

func (mockUsageBasedHandler) OnPaymentAuthorized(context.Context, usagebased.OnPaymentAuthorizedInput) (ledgertransaction.GroupReference, error) {
	return newMockLedgerTransactionGroupReference(), nil
}

func (mockUsageBasedHandler) OnPaymentSettled(context.Context, usagebased.OnPaymentSettledInput) (ledgertransaction.GroupReference, error) {
	return newMockLedgerTransactionGroupReference(), nil
}

func (mockUsageBasedHandler) OnCreditsOnlyUsageAccrued(_ context.Context, input usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
	return creditrealization.CreateAllocationInputs{
		{
			ServicePeriod:     input.Charge.Intent.ServicePeriod,
			LedgerTransaction: newMockLedgerTransactionGroupReference(),
			Amount:            input.AmountToAllocate,
		},
	}, nil
}

func (mockUsageBasedHandler) OnCreditsOnlyUsageAccruedCorrection(_ context.Context, input usagebased.CreditsOnlyUsageAccruedCorrectionInput) (creditrealization.CreateCorrectionInputs, error) {
	return lo.Map(input.Corrections, func(correction creditrealization.CorrectionRequestItem, _ int) creditrealization.CreateCorrectionInput {
		return creditrealization.CreateCorrectionInput{
			LedgerTransaction:     newMockLedgerTransactionGroupReference(),
			Amount:                correction.Amount,
			CorrectsRealizationID: correction.Allocation.ID,
		}
	}), nil
}

func newMockLedgerTransactionGroupReference() ledgertransaction.GroupReference {
	return ledgertransaction.GroupReference{
		TransactionGroupID: ulid.Make().String(),
	}
}
