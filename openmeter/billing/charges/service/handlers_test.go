package service

import (
	"context"
	"errors"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
)

var _ charges.FlatFeeHandler = (*flatFeeTestHandler)(nil)

type flatFeeTestHandler struct {
	onFlatFeeAssignedToInvoice           func(ctx context.Context, input charges.OnFlatFeeAssignedToInvoiceInput) ([]charges.CreditRealizationCreateInput, error)
	onFlatFeeStandardInvoiceUsageAccrued func(ctx context.Context, input charges.OnFlatFeeStandardInvoiceUsageAccruedInput) (charges.LedgerTransactionGroupReference, error)
	onFlatFeePaymentAuthorized           func(ctx context.Context, charge charges.FlatFeeCharge) (charges.LedgerTransactionGroupReference, error)
	onFlatFeePaymentSettled              func(ctx context.Context, charge charges.FlatFeeCharge) (charges.LedgerTransactionGroupReference, error)
	onFlatFeePaymentUncollectible        func(ctx context.Context, charge charges.FlatFeeCharge) (charges.LedgerTransactionGroupReference, error)
}

func newFlatFeeTestHandler() *flatFeeTestHandler {
	return &flatFeeTestHandler{}
}

func (h *flatFeeTestHandler) OnFlatFeeAssignedToInvoice(ctx context.Context, input charges.OnFlatFeeAssignedToInvoiceInput) ([]charges.CreditRealizationCreateInput, error) {
	if h.onFlatFeeAssignedToInvoice == nil {
		return nil, errors.New("onFlatFeeAssignedToInvoice is not set")
	}

	return h.onFlatFeeAssignedToInvoice(ctx, input)
}

func (h *flatFeeTestHandler) OnFlatFeeStandardInvoiceUsageAccrued(ctx context.Context, input charges.OnFlatFeeStandardInvoiceUsageAccruedInput) (charges.LedgerTransactionGroupReference, error) {
	if h.onFlatFeeStandardInvoiceUsageAccrued == nil {
		return charges.LedgerTransactionGroupReference{}, errors.New("onFlatFeeStandardInvoiceUsageAccrued is not set")
	}

	return h.onFlatFeeStandardInvoiceUsageAccrued(ctx, input)
}

func (h *flatFeeTestHandler) OnFlatFeePaymentAuthorized(ctx context.Context, charge charges.FlatFeeCharge) (charges.LedgerTransactionGroupReference, error) {
	if h.onFlatFeePaymentAuthorized == nil {
		return charges.LedgerTransactionGroupReference{}, errors.New("onFlatFeePaymentAuthorized is not set")
	}

	return h.onFlatFeePaymentAuthorized(ctx, charge)
}

func (h *flatFeeTestHandler) OnFlatFeePaymentSettled(ctx context.Context, charge charges.FlatFeeCharge) (charges.LedgerTransactionGroupReference, error) {
	if h.onFlatFeePaymentSettled == nil {
		return charges.LedgerTransactionGroupReference{}, errors.New("onFlatFeePaymentSettled is not set")
	}

	return h.onFlatFeePaymentSettled(ctx, charge)
}

func (h *flatFeeTestHandler) OnFlatFeePaymentUncollectible(ctx context.Context, charge charges.FlatFeeCharge) (charges.LedgerTransactionGroupReference, error) {
	if h.onFlatFeePaymentUncollectible == nil {
		return charges.LedgerTransactionGroupReference{}, errors.New("onFlatFeePaymentUncollectible is not set")
	}

	return h.onFlatFeePaymentUncollectible(ctx, charge)
}

func (h *flatFeeTestHandler) Reset() {
	*h = flatFeeTestHandler{}
}

var _ charges.CreditPurchaseHandler = (*creditPurchaseTestHandler)(nil)

type creditPurchaseTestHandler struct {
	onPromotionalCreditPurchase       func(ctx context.Context, charge charges.CreditPurchaseCharge) (charges.LedgerTransactionGroupReference, error)
	onCreditPurchaseInitiated         func(ctx context.Context, charge charges.CreditPurchaseCharge) (charges.LedgerTransactionGroupReference, error)
	onCreditPurchasePaymentAuthorized func(ctx context.Context, charge charges.CreditPurchaseCharge) (charges.LedgerTransactionGroupReference, error)
	onCreditPurchasePaymentSettled    func(ctx context.Context, charge charges.CreditPurchaseCharge) (charges.LedgerTransactionGroupReference, error)
}

func newCreditPurchaseTestHandler() *creditPurchaseTestHandler {
	return &creditPurchaseTestHandler{}
}

func (h *creditPurchaseTestHandler) OnPromotionalCreditPurchase(ctx context.Context, charge charges.CreditPurchaseCharge) (charges.LedgerTransactionGroupReference, error) {
	if h.onPromotionalCreditPurchase == nil {
		return charges.LedgerTransactionGroupReference{}, errors.New("onPromotionalCreditPurchase is not set")
	}

	return h.onPromotionalCreditPurchase(ctx, charge)
}

func (h *creditPurchaseTestHandler) OnCreditPurchaseInitiated(ctx context.Context, charge charges.CreditPurchaseCharge) (charges.LedgerTransactionGroupReference, error) {
	if h.onCreditPurchaseInitiated == nil {
		return charges.LedgerTransactionGroupReference{}, errors.New("onCreditPurchaseInitiated is not set")
	}

	return h.onCreditPurchaseInitiated(ctx, charge)
}

func (h *creditPurchaseTestHandler) OnCreditPurchasePaymentAuthorized(ctx context.Context, charge charges.CreditPurchaseCharge) (charges.LedgerTransactionGroupReference, error) {
	if h.onCreditPurchasePaymentAuthorized == nil {
		return charges.LedgerTransactionGroupReference{}, errors.New("onCreditPurchasePaymentAuthorized is not set")
	}

	return h.onCreditPurchasePaymentAuthorized(ctx, charge)
}

func (h *creditPurchaseTestHandler) OnCreditPurchasePaymentSettled(ctx context.Context, charge charges.CreditPurchaseCharge) (charges.LedgerTransactionGroupReference, error) {
	if h.onCreditPurchasePaymentSettled == nil {
		return charges.LedgerTransactionGroupReference{}, errors.New("onCreditPurchasePaymentSettled is not set")
	}

	return h.onCreditPurchasePaymentSettled(ctx, charge)
}

func (h *creditPurchaseTestHandler) Reset() {
	*h = creditPurchaseTestHandler{}
}

// helpers

type countedLedgerTransactionCallback[T any] struct {
	nrInvocations int
	id            string
}

type assertFunc[T any] func(*testing.T, T)

func newCountedLedgerTransactionCallback[T any]() *countedLedgerTransactionCallback[T] {
	return &countedLedgerTransactionCallback[T]{
		nrInvocations: 0,
		id:            ulid.Make().String(),
	}
}

func (c *countedLedgerTransactionCallback[T]) Handler(t *testing.T, asserts ...assertFunc[T]) func(ctx context.Context, t T) (charges.LedgerTransactionGroupReference, error) {
	return func(ctx context.Context, arg T) (charges.LedgerTransactionGroupReference, error) {
		c.nrInvocations++
		for _, assert := range asserts {
			assert(t, arg)
		}
		return charges.LedgerTransactionGroupReference{
			TransactionGroupID: c.id,
		}, nil
	}
}
