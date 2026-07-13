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
	"github.com/openmeterio/openmeter/pkg/currencyx"
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

// This acknowledges FBO-backed usage on the ledger by consuming value from prioritized
// customer FBO subaccounts and moving it into customer_accrued. This is NOT revenue recognition.
func (h *flatFeeHandler) OnAllocateCredits(ctx context.Context, input flatfee.OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	if input.PreTaxAmountToAllocate.IsZero() {
		return nil, nil
	}

	intent := input.Charge.Intent
	taxConfig := intent.GetTaxConfig()

	if err := validateSettlementMode(
		intent.GetSettlementMode(),
		productcatalog.CreditThenInvoiceSettlementMode,
		productcatalog.CreditOnlySettlementMode,
	); err != nil {
		return nil, fmt.Errorf("allocate credits: %w", err)
	}

	realizations, err := h.collector.CollectToAccrued(ctx, collector.CollectToAccruedInput{
		Namespace:         input.Charge.Namespace,
		ChargeID:          input.Charge.ID,
		CustomerID:        intent.GetCustomerID(),
		Annotations:       chargeAnnotationsForFlatFeeCharge(input.Charge),
		BookedAt:          input.BookedAt,
		SourceBalanceAsOf: intent.GetEffectiveInvoiceAt(),
		Currency:          intent.GetCurrency(),
		TaxCode:           lo.ToPtr(taxConfig.TaxCodeID),
		TaxBehavior:       (*ledger.TaxBehavior)(taxConfig.Behavior),
		SettlementMode:    intent.GetSettlementMode(),
		ServicePeriod:     input.ServicePeriod,
		FeatureKey:        intent.GetFeatureKey(),
		Amount:            input.PreTaxAmountToAllocate,
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

	intent := input.Charge.Intent
	taxConfig := intent.GetTaxConfig()

	if err := validateSettlementMode(
		intent.GetSettlementMode(),
		productcatalog.CreditThenInvoiceSettlementMode,
	); err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("invoice usage accrued: %w", err)
	}

	customerID := customer.CustomerID{
		Namespace: input.Charge.Namespace,
		ID:        intent.GetCustomerID(),
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
			At:            input.BookedAt,
			Amount:        amount,
			Currency:      intent.GetCurrency(),
			TaxCode:       lo.ToPtr(taxConfig.TaxCodeID),
			TaxBehavior:   (*ledger.TaxBehavior)(taxConfig.Behavior),
			CostBasis:     invoiceCostBasis,
			SpendChargeID: &input.Charge.ID,
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

func (h *flatFeeHandler) OnCorrectCreditAllocations(ctx context.Context, input flatfee.CorrectCreditAllocationsInput) (creditrealization.CreateCorrectionInputs, error) {
	intent := input.Charge.Intent

	currency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
		WithCode(intent.GetCurrency()).
		Build()
	if err != nil {
		return nil, fmt.Errorf("get currency calculator: %w", err)
	}

	if err := input.ValidateWith(currency); err != nil {
		return nil, err
	}

	return h.collector.CorrectCollectedAccrued(ctx, collector.CorrectCollectedAccruedInput{
		Namespace:                    input.Charge.Namespace,
		ChargeID:                     input.Charge.ID,
		CustomerID:                   intent.GetCustomerID(),
		Annotations:                  chargeAnnotationsForFlatFeeCharge(input.Charge),
		AllocateAt:                   input.BookedAt,
		Corrections:                  input.Corrections,
		LineageSegmentsByRealization: input.LineageSegmentsByRealization,
	})
}

// OnFlatFeePaymentAuthorized stages the directly-invoiced receivable as
// authorized. Revenue recognition is handled elsewhere.
func (h *flatFeeHandler) OnPaymentAuthorized(ctx context.Context, input flatfee.OnPaymentAuthorizedInput) (ledgertransaction.GroupReference, error) {
	if err := input.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	if input.Amount.IsZero() {
		return ledgertransaction.GroupReference{}, nil
	}

	intent := input.Charge.Intent

	customerID := customer.CustomerID{
		Namespace: input.Charge.Namespace,
		ID:        intent.GetCustomerID(),
	}
	annotations := chargeAnnotationsForFlatFeeCharge(input.Charge)

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.deps,
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  input.Charge.Namespace,
		},
		transactions.AuthorizeCustomerReceivablePaymentTemplate{
			At:            input.EventAt,
			Amount:        input.Amount,
			Currency:      intent.GetCurrency(),
			CostBasis:     invoiceCostBasis,
			SpendChargeID: &input.Charge.ID,
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

func (h *flatFeeHandler) OnPaymentSettled(ctx context.Context, input flatfee.OnPaymentSettledInput) (ledgertransaction.GroupReference, error) {
	if err := input.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	if !input.Amount.IsPositive() {
		return ledgertransaction.GroupReference{}, nil
	}

	intent := input.Charge.Intent

	customerID := customer.CustomerID{
		Namespace: input.Charge.Namespace,
		ID:        intent.GetCustomerID(),
	}
	annotations := chargeAnnotationsForFlatFeeCharge(input.Charge)

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.deps,
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  input.Charge.Namespace,
		},
		transactions.SettleCustomerReceivableFromPaymentTemplate{
			At:            input.EventAt,
			Amount:        input.Amount,
			Currency:      intent.GetCurrency(),
			CostBasis:     invoiceCostBasis,
			SpendChargeID: &input.Charge.ID,
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
