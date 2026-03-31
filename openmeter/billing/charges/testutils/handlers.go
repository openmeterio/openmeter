package testutils

import (
	"context"

	"github.com/oklog/ulid/v2"

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

func (mockFlatFeeHandler) OnAssignedToInvoice(context.Context, flatfee.OnAssignedToInvoiceInput) ([]creditrealization.CreateInput, error) {
	return nil, nil
}

func (mockFlatFeeHandler) OnInvoiceUsageAccrued(context.Context, flatfee.OnInvoiceUsageAccruedInput) (ledgertransaction.GroupReference, error) {
	return newMockLedgerTransactionGroupReference(), nil
}

func (mockFlatFeeHandler) OnCreditsOnlyUsageAccrued(_ context.Context, input flatfee.OnCreditsOnlyUsageAccruedInput) ([]creditrealization.CreateInput, error) {
	return []creditrealization.CreateInput{
		{
			ServicePeriod:     input.Charge.Intent.ServicePeriod,
			LedgerTransaction: newMockLedgerTransactionGroupReference(),
			Amount:            input.AmountToAllocate,
		},
	}, nil
}

func (mockFlatFeeHandler) OnPaymentAuthorized(context.Context, flatfee.Charge) (ledgertransaction.GroupReference, error) {
	return newMockLedgerTransactionGroupReference(), nil
}

func (mockFlatFeeHandler) OnPaymentSettled(context.Context, flatfee.Charge) (ledgertransaction.GroupReference, error) {
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

func (mockUsageBasedHandler) OnCollectionStarted(_ context.Context, input usagebased.AllocateCreditsInput) (creditrealization.CreateInputs, error) {
	return []creditrealization.CreateInput{
		{
			ServicePeriod:     input.Charge.Intent.ServicePeriod,
			LedgerTransaction: newMockLedgerTransactionGroupReference(),
			Amount:            input.AmountToAllocate,
		},
	}, nil
}

func (mockUsageBasedHandler) OnCollectionFinalized(_ context.Context, input usagebased.AllocateCreditsInput) (creditrealization.CreateInputs, error) {
	return []creditrealization.CreateInput{
		{
			ServicePeriod:     input.Charge.Intent.ServicePeriod,
			LedgerTransaction: newMockLedgerTransactionGroupReference(),
			Amount:            input.AmountToAllocate,
		},
	}, nil
}

func newMockLedgerTransactionGroupReference() ledgertransaction.GroupReference {
	return ledgertransaction.GroupReference{
		TransactionGroupID: ulid.Make().String(),
	}
}
