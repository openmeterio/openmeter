package service

import (
	"context"
	"errors"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
)

var _ flatfee.Handler = (*flatFeeTestHandler)(nil)

type flatFeeTestHandler struct {
	onAssignedToInvoice                 func(ctx context.Context, input flatfee.OnAssignedToInvoiceInput) (creditrealization.CreateAllocationInputs, error)
	onInvoiceUsageAccrued               func(ctx context.Context, input flatfee.OnInvoiceUsageAccruedInput) (ledgertransaction.GroupReference, error)
	onCreditsOnlyUsageAccrued           func(ctx context.Context, input flatfee.OnCreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error)
	onCreditsOnlyUsageAccruedCorrection func(ctx context.Context, input flatfee.CreditsOnlyUsageAccruedCorrectionInput) (creditrealization.CreateCorrectionInputs, error)
	onPaymentAuthorized                 func(ctx context.Context, charge flatfee.Charge) (ledgertransaction.GroupReference, error)
	onPaymentSettled                    func(ctx context.Context, charge flatfee.Charge) (ledgertransaction.GroupReference, error)
	onPaymentUncollectible              func(ctx context.Context, charge flatfee.Charge) (ledgertransaction.GroupReference, error)
}

func newFlatFeeTestHandler() *flatFeeTestHandler {
	return &flatFeeTestHandler{}
}

func (h *flatFeeTestHandler) OnAssignedToInvoice(ctx context.Context, input flatfee.OnAssignedToInvoiceInput) (creditrealization.CreateAllocationInputs, error) {
	if h.onAssignedToInvoice == nil {
		return nil, errors.New("onAssignedToInvoice is not set")
	}

	return h.onAssignedToInvoice(ctx, input)
}

func (h *flatFeeTestHandler) OnInvoiceUsageAccrued(ctx context.Context, input flatfee.OnInvoiceUsageAccruedInput) (ledgertransaction.GroupReference, error) {
	if h.onInvoiceUsageAccrued == nil {
		return ledgertransaction.GroupReference{}, errors.New("onInvoiceUsageAccrued is not set")
	}

	return h.onInvoiceUsageAccrued(ctx, input)
}

func (h *flatFeeTestHandler) OnCreditsOnlyUsageAccrued(ctx context.Context, input flatfee.OnCreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
	if h.onCreditsOnlyUsageAccrued == nil {
		return nil, errors.New("onCreditsOnlyUsageAccrued is not set")
	}

	return h.onCreditsOnlyUsageAccrued(ctx, input)
}

func (h *flatFeeTestHandler) OnCreditsOnlyUsageAccruedCorrection(ctx context.Context, input flatfee.CreditsOnlyUsageAccruedCorrectionInput) (creditrealization.CreateCorrectionInputs, error) {
	if h.onCreditsOnlyUsageAccruedCorrection == nil {
		return nil, errors.New("onCreditsOnlyUsageAccruedCorrection is not set")
	}

	return h.onCreditsOnlyUsageAccruedCorrection(ctx, input)
}

func (h *flatFeeTestHandler) OnPaymentAuthorized(ctx context.Context, charge flatfee.Charge) (ledgertransaction.GroupReference, error) {
	if h.onPaymentAuthorized == nil {
		return ledgertransaction.GroupReference{}, errors.New("onPaymentAuthorized is not set")
	}

	return h.onPaymentAuthorized(ctx, charge)
}

func (h *flatFeeTestHandler) OnPaymentSettled(ctx context.Context, charge flatfee.Charge) (ledgertransaction.GroupReference, error) {
	if h.onPaymentSettled == nil {
		return ledgertransaction.GroupReference{}, errors.New("onPaymentSettled is not set")
	}

	return h.onPaymentSettled(ctx, charge)
}

func (h *flatFeeTestHandler) OnPaymentUncollectible(ctx context.Context, charge flatfee.Charge) (ledgertransaction.GroupReference, error) {
	if h.onPaymentUncollectible == nil {
		return ledgertransaction.GroupReference{}, errors.New("onPaymentUncollectible is not set")
	}

	return h.onPaymentUncollectible(ctx, charge)
}

func (h *flatFeeTestHandler) Reset() {
	*h = flatFeeTestHandler{}
}

var _ creditpurchase.Handler = (*creditPurchaseTestHandler)(nil)

type creditPurchaseTestHandler struct {
	onPromotionalCreditPurchase       func(ctx context.Context, charge creditpurchase.Charge) (ledgertransaction.GroupReference, error)
	onCreditPurchaseInitiated         func(ctx context.Context, charge creditpurchase.Charge) (ledgertransaction.GroupReference, error)
	onCreditPurchasePaymentAuthorized func(ctx context.Context, charge creditpurchase.Charge) (ledgertransaction.GroupReference, error)
	onCreditPurchasePaymentSettled    func(ctx context.Context, charge creditpurchase.Charge) (ledgertransaction.GroupReference, error)
}

func newCreditPurchaseTestHandler() *creditPurchaseTestHandler {
	return &creditPurchaseTestHandler{}
}

func (h *creditPurchaseTestHandler) OnPromotionalCreditPurchase(ctx context.Context, charge creditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	if h.onPromotionalCreditPurchase == nil {
		return ledgertransaction.GroupReference{}, errors.New("onPromotionalCreditPurchase is not set")
	}

	return h.onPromotionalCreditPurchase(ctx, charge)
}

func (h *creditPurchaseTestHandler) OnCreditPurchaseInitiated(ctx context.Context, charge creditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	if h.onCreditPurchaseInitiated == nil {
		return ledgertransaction.GroupReference{}, errors.New("onCreditPurchaseInitiated is not set")
	}

	return h.onCreditPurchaseInitiated(ctx, charge)
}

func (h *creditPurchaseTestHandler) OnCreditPurchasePaymentAuthorized(ctx context.Context, charge creditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	if h.onCreditPurchasePaymentAuthorized == nil {
		return ledgertransaction.GroupReference{}, errors.New("onCreditPurchasePaymentAuthorized is not set")
	}

	return h.onCreditPurchasePaymentAuthorized(ctx, charge)
}

func (h *creditPurchaseTestHandler) OnCreditPurchasePaymentSettled(ctx context.Context, charge creditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	if h.onCreditPurchasePaymentSettled == nil {
		return ledgertransaction.GroupReference{}, errors.New("onCreditPurchasePaymentSettled is not set")
	}

	return h.onCreditPurchasePaymentSettled(ctx, charge)
}

func (h *creditPurchaseTestHandler) Reset() {
	*h = creditPurchaseTestHandler{}
}

var _ usagebased.Handler = (*usageBasedTestHandler)(nil)

type usageBasedTestHandler struct {
	onInvoiceUsageAccrued               func(ctx context.Context, input usagebased.OnInvoiceUsageAccruedInput) (ledgertransaction.GroupReference, error)
	onPaymentAuthorized                 func(ctx context.Context, input usagebased.OnPaymentAuthorizedInput) (ledgertransaction.GroupReference, error)
	onPaymentSettled                    func(ctx context.Context, input usagebased.OnPaymentSettledInput) (ledgertransaction.GroupReference, error)
	onCreditsOnlyUsageAccrued           func(ctx context.Context, input usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error)
	onCreditsOnlyUsageAccruedCorrection func(ctx context.Context, input usagebased.CreditsOnlyUsageAccruedCorrectionInput) (creditrealization.CreateCorrectionInputs, error)
}

func newUsageBasedTestHandler() *usageBasedTestHandler {
	return &usageBasedTestHandler{}
}

func (h *usageBasedTestHandler) OnInvoiceUsageAccrued(ctx context.Context, input usagebased.OnInvoiceUsageAccruedInput) (ledgertransaction.GroupReference, error) {
	if h.onInvoiceUsageAccrued == nil {
		return ledgertransaction.GroupReference{}, errors.New("onInvoiceUsageAccrued is not set")
	}

	return h.onInvoiceUsageAccrued(ctx, input)
}

func (h *usageBasedTestHandler) OnPaymentAuthorized(ctx context.Context, input usagebased.OnPaymentAuthorizedInput) (ledgertransaction.GroupReference, error) {
	if h.onPaymentAuthorized == nil {
		return ledgertransaction.GroupReference{}, errors.New("onPaymentAuthorized is not set")
	}

	return h.onPaymentAuthorized(ctx, input)
}

func (h *usageBasedTestHandler) OnPaymentSettled(ctx context.Context, input usagebased.OnPaymentSettledInput) (ledgertransaction.GroupReference, error) {
	if h.onPaymentSettled == nil {
		return ledgertransaction.GroupReference{}, errors.New("onPaymentSettled is not set")
	}

	return h.onPaymentSettled(ctx, input)
}

func (h *usageBasedTestHandler) OnCreditsOnlyUsageAccrued(ctx context.Context, input usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
	if h.onCreditsOnlyUsageAccrued == nil {
		return nil, errors.New("onCreditsOnlyUsageAccrued is not set")
	}

	return h.onCreditsOnlyUsageAccrued(ctx, input)
}

func (h *usageBasedTestHandler) OnCreditsOnlyUsageAccruedCorrection(ctx context.Context, input usagebased.CreditsOnlyUsageAccruedCorrectionInput) (creditrealization.CreateCorrectionInputs, error) {
	if h.onCreditsOnlyUsageAccruedCorrection == nil {
		return nil, errors.New("onCreditsOnlyUsageAccruedCorrection is not set")
	}

	return h.onCreditsOnlyUsageAccruedCorrection(ctx, input)
}

func (h *usageBasedTestHandler) Reset() {
	*h = usageBasedTestHandler{}
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

type countedCreditAllocationCallback[T any] struct {
	nrInvocations int
	id            string
}

func newCountedCreditAllocationCallback[T any]() *countedCreditAllocationCallback[T] {
	return &countedCreditAllocationCallback[T]{
		nrInvocations: 0,
		id:            ulid.Make().String(),
	}
}

func newCappedCreditAllocator(availableCredits float64) (func(ctx context.Context, input usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error), *alpacadecimal.Decimal) {
	remainingCredits := alpacadecimal.NewFromFloat(availableCredits)

	return func(ctx context.Context, input usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
		amount := input.AmountToAllocate
		if amount.GreaterThan(remainingCredits) {
			amount = remainingCredits
		}

		if amount.IsZero() {
			return nil, nil
		}

		remainingCredits = remainingCredits.Sub(amount)

		return creditrealization.CreateAllocationInputs{
			{
				ServicePeriod: input.Charge.Intent.ServicePeriod,
				Amount:        amount,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: ulid.Make().String(),
				},
			},
		}, nil
	}, &remainingCredits
}

func (c *countedLedgerTransactionCallback[T]) Handler(t *testing.T, asserts ...assertFunc[T]) func(ctx context.Context, t T) (ledgertransaction.GroupReference, error) {
	return func(ctx context.Context, arg T) (ledgertransaction.GroupReference, error) {
		c.nrInvocations++
		for _, assert := range asserts {
			assert(t, arg)
		}
		return ledgertransaction.GroupReference{
			TransactionGroupID: c.id,
		}, nil
	}
}

func (c *countedCreditAllocationCallback[T]) Handler(t *testing.T, allocations func(T, ledgertransaction.GroupReference) creditrealization.CreateAllocationInputs, asserts ...assertFunc[T]) func(ctx context.Context, t T) (creditrealization.CreateAllocationInputs, error) {
	return func(ctx context.Context, arg T) (creditrealization.CreateAllocationInputs, error) {
		c.nrInvocations++
		for _, assert := range asserts {
			assert(t, arg)
		}

		return allocations(arg, ledgertransaction.GroupReference{
			TransactionGroupID: c.id,
		}), nil
	}
}
