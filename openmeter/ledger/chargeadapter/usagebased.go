package chargeadapter

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
)

// usageBasedHandler maps usage-based credit lifecycle events to ledger transaction templates.
type usageBasedHandler struct {
	ledger    ledger.Ledger
	deps      transactions.ResolverDependencies
	collector collector.Service
}

var _ usagebased.Handler = (*usageBasedHandler)(nil)

func NewUsageBasedHandler(
	ledger ledger.Ledger,
	deps transactions.ResolverDependencies,
	collectorService collector.Service,
) usagebased.Handler {
	return &usageBasedHandler{
		ledger:    ledger,
		deps:      deps,
		collector: collectorService,
	}
}

func (h *usageBasedHandler) OnInvoiceUsageAccrued(ctx context.Context, input usagebased.OnInvoiceUsageAccruedInput) (ledgertransaction.GroupReference, error) {
	if err := input.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	amount := input.Amount
	if amount.IsZero() {
		return ledgertransaction.GroupReference{}, nil
	}

	if err := validateSettlementMode(
		input.Charge.Intent.SettlementMode,
		productcatalog.InvoiceOnlySettlementMode,
		productcatalog.CreditThenInvoiceSettlementMode,
	); err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("invoice usage accrued: %w", err)
	}

	customerID := customer.CustomerID{
		Namespace: input.Charge.Namespace,
		ID:        input.Charge.Intent.CustomerID,
	}

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.deps,
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  input.Charge.Namespace,
		},
		transactions.TransferCustomerReceivableToAccruedTemplate{
			At:        input.Charge.Intent.InvoiceAt,
			Amount:    amount,
			Currency:  input.Charge.Intent.Currency,
			CostBasis: invoiceCostBasis,
		},
	)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("resolve transactions: %w", err)
	}

	transactionGroup, err := h.ledger.CommitGroup(ctx, transactions.GroupInputs(
		input.Charge.Namespace,
		chargeAnnotationsForUsageBasedCharge(input.Charge),
		inputs...,
	))
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("commit ledger transaction group: %w", err)
	}

	return ledgertransaction.GroupReference{
		TransactionGroupID: transactionGroup.ID().ID,
	}, nil
}

func (h *usageBasedHandler) OnPaymentAuthorized(ctx context.Context, input usagebased.OnPaymentAuthorizedInput) (ledgertransaction.GroupReference, error) {
	if err := input.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}
	eventTime := clock.Now()

	if err := validateSettlementMode(
		input.Charge.Intent.SettlementMode,
		productcatalog.InvoiceOnlySettlementMode,
		productcatalog.CreditThenInvoiceSettlementMode,
	); err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("payment authorized: %w", err)
	}

	receivableReplenishment := alpacadecimal.Zero
	if input.Run.InvoiceUsage != nil {
		receivableReplenishment = input.Run.InvoiceUsage.Totals.Total
	}

	if receivableReplenishment.IsZero() {
		return ledgertransaction.GroupReference{}, nil
	}

	customerID := customer.CustomerID{
		Namespace: input.Charge.Namespace,
		ID:        input.Charge.Intent.CustomerID,
	}
	annotations := chargeAnnotationsForUsageBasedCharge(input.Charge)

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.deps,
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  input.Charge.Namespace,
		},
		transactions.FundCustomerReceivableTemplate{
			At:        eventTime,
			Amount:    receivableReplenishment,
			Currency:  input.Charge.Intent.Currency,
			CostBasis: invoiceCostBasis,
		},
	)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("resolve transactions: %w", err)
	}

	for i, txInput := range inputs {
		if txInput != nil {
			inputs[i] = transactions.WithAnnotations(txInput, annotations)
		}
	}

	transactionGroup, err := h.ledger.CommitGroup(ctx, transactions.GroupInputs(
		input.Charge.Namespace,
		annotations,
		inputs...,
	))
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("commit ledger transaction group: %w", err)
	}

	return ledgertransaction.GroupReference{
		TransactionGroupID: transactionGroup.ID().ID,
	}, nil
}

func (h *usageBasedHandler) OnPaymentSettled(ctx context.Context, input usagebased.OnPaymentSettledInput) (ledgertransaction.GroupReference, error) {
	if err := input.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}
	eventTime := clock.Now()

	if err := validateSettlementMode(
		input.Charge.Intent.SettlementMode,
		productcatalog.InvoiceOnlySettlementMode,
		productcatalog.CreditThenInvoiceSettlementMode,
	); err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("payment settled: %w", err)
	}

	if input.Run.InvoiceUsage == nil || !input.Run.InvoiceUsage.Totals.Total.IsPositive() {
		return ledgertransaction.GroupReference{}, nil
	}

	customerID := customer.CustomerID{
		Namespace: input.Charge.Namespace,
		ID:        input.Charge.Intent.CustomerID,
	}
	annotations := chargeAnnotationsForUsageBasedCharge(input.Charge)

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.deps,
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  input.Charge.Namespace,
		},
		transactions.SettleCustomerReceivablePaymentTemplate{
			At:        eventTime,
			Amount:    input.Run.InvoiceUsage.Totals.Total,
			Currency:  input.Charge.Intent.Currency,
			CostBasis: invoiceCostBasis,
		},
	)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("resolve transactions: %w", err)
	}

	for i, txInput := range inputs {
		if txInput != nil {
			inputs[i] = transactions.WithAnnotations(txInput, annotations)
		}
	}

	transactionGroup, err := h.ledger.CommitGroup(ctx, transactions.GroupInputs(
		input.Charge.Namespace,
		annotations,
		inputs...,
	))
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("commit ledger transaction group: %w", err)
	}

	return ledgertransaction.GroupReference{
		TransactionGroupID: transactionGroup.ID().ID,
	}, nil
}

func (h *usageBasedHandler) OnCreditsOnlyUsageAccrued(ctx context.Context, input usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	if input.AmountToAllocate.IsZero() {
		return nil, nil
	}

	if err := validateSettlementMode(
		input.Charge.Intent.SettlementMode,
		productcatalog.CreditOnlySettlementMode,
		productcatalog.CreditThenInvoiceSettlementMode,
	); err != nil {
		return nil, fmt.Errorf("credits only usage accrued: %w", err)
	}

	realizations, err := h.collector.CollectToAccrued(ctx, collector.CollectToAccruedInput{
		Namespace:      input.Charge.Namespace,
		ChargeID:       input.Charge.ID,
		CustomerID:     input.Charge.Intent.CustomerID,
		Annotations:    chargeAnnotationsForUsageBasedCharge(input.Charge),
		At:             input.AllocateAt,
		Currency:       input.Charge.Intent.Currency,
		SettlementMode: input.Charge.Intent.SettlementMode,
		ServicePeriod:  input.Charge.Intent.ServicePeriod,
		Amount:         input.AmountToAllocate,
	})
	if err != nil {
		return nil, err
	}
	if len(realizations) == 0 {
		return nil, nil
	}

	return realizations, nil
}

func (h *usageBasedHandler) OnCreditsOnlyUsageAccruedCorrection(ctx context.Context, input usagebased.CreditsOnlyUsageAccruedCorrectionInput) (creditrealization.CreateCorrectionInputs, error) {
	if err := input.Charge.Validate(); err != nil {
		return nil, fmt.Errorf("charge: %w", err)
	}

	if err := input.Run.Validate(); err != nil {
		return nil, fmt.Errorf("run: %w", err)
	}

	if input.AllocateAt.IsZero() {
		return nil, fmt.Errorf("allocate at is required")
	}

	if err := validateSettlementMode(
		input.Charge.Intent.SettlementMode,
		productcatalog.CreditOnlySettlementMode,
		productcatalog.CreditThenInvoiceSettlementMode,
	); err != nil {
		return nil, fmt.Errorf("credits only usage accrued correction: %w", err)
	}

	currencyCalculator, err := input.Charge.Intent.Currency.Calculator()
	if err != nil {
		return nil, fmt.Errorf("get currency calculator: %w", err)
	}

	if err := input.Corrections.ValidateWith(currencyCalculator); err != nil {
		return nil, fmt.Errorf("corrections: %w", err)
	}

	return h.collector.CorrectCollectedAccrued(ctx, collector.CorrectCollectedAccruedInput{
		Namespace:                    input.Charge.Namespace,
		ChargeID:                     input.Charge.ID,
		CustomerID:                   input.Charge.Intent.CustomerID,
		Annotations:                  chargeAnnotationsForUsageBasedCharge(input.Charge),
		AllocateAt:                   input.AllocateAt,
		Corrections:                  input.Corrections,
		LineageSegmentsByRealization: input.LineageSegmentsByRealization,
	})
}
