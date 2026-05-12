package chargeadapter

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// flatFeeHandler maps charge lifecycle events to ledger transaction templates
type flatFeeHandler struct {
	ledger    ledger.Ledger
	deps      transactions.ResolverDependencies
	collector collector.Service
}

var _ flatfee.Handler = (*flatFeeHandler)(nil)

func NewFlatFeeHandler(
	ledger ledger.Ledger,
	deps transactions.ResolverDependencies,
	collectorService collector.Service,
) flatfee.Handler {
	return &flatFeeHandler{
		ledger:    ledger,
		deps:      deps,
		collector: collectorService,
	}
}

// OnFlatFeeAssignedToInvoice is called when a flat fee is being assigned to an invoice.
// This acknowledges FBO-backed usage on the ledger by consuming value from prioritized
// customer FBO subaccounts and moving it into customer_accrued. This is NOT revenue recognition.
func (h *flatFeeHandler) OnAssignedToInvoice(ctx context.Context, input flatfee.OnAssignedToInvoiceInput) (creditrealization.CreateAllocationInputs, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	if input.PreTaxTotalAmount.IsZero() {
		return nil, nil
	}

	if err := validateSettlementMode(
		input.Charge.Intent.SettlementMode,
		productcatalog.InvoiceOnlySettlementMode,
		productcatalog.CreditThenInvoiceSettlementMode,
	); err != nil {
		return nil, fmt.Errorf("assigned to invoice: %w", err)
	}

	if input.Charge.Intent.SettlementMode == productcatalog.InvoiceOnlySettlementMode {
		return nil, nil
	}

	realizations, err := h.collector.CollectToAccrued(ctx, collector.CollectToAccruedInput{
		Namespace:      input.Charge.Namespace,
		ChargeID:       input.Charge.ID,
		CustomerID:     input.Charge.Intent.CustomerID,
		Annotations:    chargeAnnotationsForFlatFeeCharge(input.Charge),
		At:             input.Charge.Intent.InvoiceAt,
		Currency:       input.Charge.Intent.Currency,
		SettlementMode: input.Charge.Intent.SettlementMode,
		ServicePeriod:  input.ServicePeriod,
		Amount:         input.PreTaxTotalAmount,
	})
	if err != nil {
		return nil, err
	}
	if len(realizations) == 0 {
		return nil, nil
	}

	return realizations, nil
}

// OnFlatFeeStandardInvoiceUsageAccrued handles the portion of usage not covered by FBO credits.
// It acknowledges usage on the ledger by booking it against receivable and moving it into customer_accrued.
// This is NOT revenue recognition.
func (h *flatFeeHandler) OnInvoiceUsageAccrued(ctx context.Context, input flatfee.OnInvoiceUsageAccruedInput) (ledgertransaction.GroupReference, error) {
	if err := input.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	amount := input.Totals.Total
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
	annotations := chargeAnnotationsForFlatFeeCharge(input.Charge)

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

// OnCreditsOnlyUsageAccrued is called when a credit-only flat fee becomes active.
// It consumes value from prioritized customer FBO subaccounts and moves it into customer_accrued.
func (h *flatFeeHandler) OnCreditsOnlyUsageAccrued(ctx context.Context, input flatfee.OnCreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	if input.AmountToAllocate.IsZero() {
		return nil, nil
	}

	if err := validateSettlementMode(input.Charge.Intent.SettlementMode, productcatalog.CreditOnlySettlementMode); err != nil {
		return nil, fmt.Errorf("credits only usage accrued: %w", err)
	}

	realizations, err := h.collector.CollectToAccrued(ctx, collector.CollectToAccruedInput{
		Namespace:      input.Charge.Namespace,
		ChargeID:       input.Charge.ID,
		CustomerID:     input.Charge.Intent.CustomerID,
		Annotations:    chargeAnnotationsForFlatFeeCharge(input.Charge),
		At:             input.Charge.Intent.InvoiceAt,
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

func (h *flatFeeHandler) OnCreditsOnlyUsageAccruedCorrection(ctx context.Context, input flatfee.CreditsOnlyUsageAccruedCorrectionInput) (creditrealization.CreateCorrectionInputs, error) {
	currencyCalculator, err := input.Charge.Intent.Currency.Calculator()
	if err != nil {
		return nil, fmt.Errorf("get currency calculator: %w", err)
	}

	if err := input.ValidateWith(currencyCalculator); err != nil {
		return nil, err
	}

	return h.collector.CorrectCollectedAccrued(ctx, collector.CorrectCollectedAccruedInput{
		Namespace:                    input.Charge.Namespace,
		ChargeID:                     input.Charge.ID,
		CustomerID:                   input.Charge.Intent.CustomerID,
		Annotations:                  chargeAnnotationsForFlatFeeCharge(input.Charge),
		AllocateAt:                   input.AllocateAt,
		Corrections:                  input.Corrections,
		LineageSegmentsByRealization: input.LineageSegmentsByRealization,
	})
}

// OnFlatFeePaymentAuthorized stages the directly-invoiced receivable as
// authorized. Revenue recognition is handled elsewhere.
func (h *flatFeeHandler) OnPaymentAuthorized(ctx context.Context, charge flatfee.Charge) (ledgertransaction.GroupReference, error) {
	if err := charge.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	if charge.Realizations.CurrentRun == nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("current run is required for payment authorization")
	}

	receivableReplenishment := alpacadecimal.NewFromInt(0)
	if charge.Realizations.CurrentRun.AccruedUsage != nil {
		receivableReplenishment = charge.Realizations.CurrentRun.AccruedUsage.Totals.Total
	}

	if receivableReplenishment.IsZero() {
		return ledgertransaction.GroupReference{}, nil
	}

	customerID := customer.CustomerID{
		Namespace: charge.Namespace,
		ID:        charge.Intent.CustomerID,
	}
	annotations := chargeAnnotationsForFlatFeeCharge(charge)

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.deps,
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  charge.Namespace,
		},
		transactions.AuthorizeCustomerReceivablePaymentTemplate{
			At:        charge.Intent.InvoiceAt,
			Amount:    receivableReplenishment,
			Currency:  charge.Intent.Currency,
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
		charge.Namespace,
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

func (h *flatFeeHandler) OnPaymentSettled(ctx context.Context, charge flatfee.Charge) (ledgertransaction.GroupReference, error) {
	if err := charge.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	if charge.Realizations.CurrentRun == nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("current run is required for payment settled")
	}

	if charge.Realizations.CurrentRun.AccruedUsage == nil || !charge.Realizations.CurrentRun.AccruedUsage.Totals.Total.IsPositive() {
		return ledgertransaction.GroupReference{}, nil
	}

	customerID := customer.CustomerID{
		Namespace: charge.Namespace,
		ID:        charge.Intent.CustomerID,
	}
	annotations := chargeAnnotationsForFlatFeeCharge(charge)

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.deps,
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  charge.Namespace,
		},
		transactions.SettleCustomerReceivableFromPaymentTemplate{
			At:        charge.Intent.InvoiceAt,
			Amount:    charge.Realizations.CurrentRun.AccruedUsage.Totals.Total,
			Currency:  charge.Intent.Currency,
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
		charge.Namespace,
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

// OnFlatFeePaymentUncollectible is not yet implemented.
// The reversal/write-off accounting flow will be added later.
func (h *flatFeeHandler) OnPaymentUncollectible(_ context.Context, _ flatfee.Charge) (ledgertransaction.GroupReference, error) {
	return ledgertransaction.GroupReference{}, fmt.Errorf("flat fee uncollectible write-off is not yet implemented")
}

func validateSettlementMode(actual productcatalog.SettlementMode, allowed ...productcatalog.SettlementMode) error {
	for _, candidate := range allowed {
		if actual == candidate {
			return nil
		}
	}

	return fmt.Errorf("unsupported settlement mode %q", actual)
}

var invoiceCostBasis = lo.ToPtr(alpacadecimal.NewFromInt(1))
