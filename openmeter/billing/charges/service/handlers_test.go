package service

import (
	"context"
	"errors"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
)

var _ charges.FlatFeeHandler = (*flatFeeTestHandler)(nil)

type flatFeeTestHandler struct {
	t *testing.T

	onFlatFeeAssignedToInvoice    func(ctx context.Context, input charges.OnFlatFeeAssignedToInvoiceInput) ([]charges.CreditRealizationCreateInput, error)
	onFlatFeePaymentAuthorized    func(ctx context.Context, charge charges.FlatFeeCharge) (charges.LedgerTransactionGroupReference, error)
	onFlatFeePaymentSettled       func(ctx context.Context, charge charges.FlatFeeCharge) (charges.LedgerTransactionGroupReference, error)
	onFlatFeePaymentUncollectible func(ctx context.Context, charge charges.FlatFeeCharge) (charges.LedgerTransactionGroupReference, error)
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
