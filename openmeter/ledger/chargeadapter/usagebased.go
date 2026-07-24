package chargeadapter

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
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

	if input.Charge.Intent.GetCurrency().IsCustom() {
		return ledgertransaction.GroupReference{}, fmt.Errorf("usage based charge with custom currency: %w", meta.ErrCustomCurrencyNotSupported)
	}

	amount := input.Amount
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
			Currency:      intent.GetCurrency().GetCode(),
			TaxCode:       lo.ToPtr(taxConfig.TaxCodeID),
			TaxBehavior:   (*ledger.TaxBehavior)(taxConfig.Behavior),
			CostBasis:     invoiceCostBasis,
			SpendChargeID: &input.Charge.ID,
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

	intent := input.Charge.Intent

	if intent.GetCurrency().IsCustom() {
		// TODO[implement]: FiatAmount contains the amount paid in the fiat currency.
		return ledgertransaction.GroupReference{}, fmt.Errorf("usage based charge with custom currency: %w", meta.ErrCustomCurrencyNotSupported)
	}

	if err := validateSettlementMode(
		intent.GetSettlementMode(),
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
		ID:        intent.GetCustomerID(),
	}
	annotations := chargeAnnotationsForUsageBasedCharge(input.Charge)

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.deps,
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  input.Charge.Namespace,
		},
		transactions.AuthorizeCustomerReceivablePaymentTemplate{
			At:            input.EventAt,
			Amount:        receivableReplenishment,
			Currency:      intent.GetCurrency().GetCode(),
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

func (h *usageBasedHandler) OnCustomCurrencyOverageAccrued(ctx context.Context, input usagebased.OnCustomCurrencyOverageAccruedInput) (usagebased.OnCustomCurrencyOverageAccruedResult, error) {
	if err := input.Validate(); err != nil {
		return usagebased.OnCustomCurrencyOverageAccruedResult{}, err
	}

	return usagebased.OnCustomCurrencyOverageAccruedResult{}, fmt.Errorf("implement OnCustomCurrencyOverageAccrued: %w", meta.ErrCustomCurrencyNotSupported)
}

func (h *usageBasedHandler) OnPaymentSettled(ctx context.Context, input usagebased.OnPaymentSettledInput) (ledgertransaction.GroupReference, error) {
	if err := input.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	intent := input.Charge.Intent

	if input.Charge.Intent.GetCurrency().IsCustom() {
		// TODO[implement]: FiatAmount contains the amount paid in the fiat currency.
		return ledgertransaction.GroupReference{}, fmt.Errorf("usage based charge with custom currency: %w", meta.ErrCustomCurrencyNotSupported)
	}

	if err := validateSettlementMode(
		intent.GetSettlementMode(),
		productcatalog.CreditThenInvoiceSettlementMode,
	); err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("payment settled: %w", err)
	}

	if input.Run.InvoiceUsage == nil || !input.Run.InvoiceUsage.Totals.Total.IsPositive() {
		return ledgertransaction.GroupReference{}, nil
	}

	customerID := customer.CustomerID{
		Namespace: input.Charge.Namespace,
		ID:        intent.GetCustomerID(),
	}
	annotations := chargeAnnotationsForUsageBasedCharge(input.Charge)

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.deps,
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  input.Charge.Namespace,
		},
		transactions.SettleCustomerReceivableFromPaymentTemplate{
			At:            input.EventAt,
			Amount:        input.Run.InvoiceUsage.Totals.Total,
			Currency:      intent.GetCurrency().GetCode(),
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

func (h *usageBasedHandler) OnCreditsOnlyUsageAccrued(ctx context.Context, input usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	if input.AmountToAllocate.IsZero() {
		return nil, nil
	}

	intent := input.Charge.Intent
	taxConfig := intent.GetTaxConfig()

	if intent.GetCurrency().IsCustom() {
		return nil, fmt.Errorf("usage based charge with custom currency: %w", meta.ErrCustomCurrencyNotSupported)
	}

	if err := validateSettlementMode(
		intent.GetSettlementMode(),
		productcatalog.CreditOnlySettlementMode,
		productcatalog.CreditThenInvoiceSettlementMode,
	); err != nil {
		return nil, fmt.Errorf("credits only usage accrued: %w", err)
	}

	realizations, err := h.collector.CollectToAccrued(ctx, collector.CollectToAccruedInput{
		Namespace:         input.Charge.Namespace,
		ChargeID:          input.Charge.ID,
		CustomerID:        intent.GetCustomerID(),
		Annotations:       chargeAnnotationsForUsageBasedCharge(input.Charge),
		BookedAt:          input.BookedAt,
		SourceBalanceAsOf: input.BookedAt,
		Currency:          intent.GetCurrency().GetCode(),
		FeatureKey:        intent.GetFeatureKey(),
		TaxCode:           lo.ToPtr(taxConfig.TaxCodeID),
		TaxBehavior:       (*ledger.TaxBehavior)(taxConfig.Behavior),
		SettlementMode:    intent.GetSettlementMode(),
		ServicePeriod:     intent.GetEffectiveServicePeriod(),
		Amount:            input.AmountToAllocate,
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
	intent := input.Charge.Intent

	if err := validateSettlementMode(
		intent.GetSettlementMode(),
		productcatalog.CreditOnlySettlementMode,
		productcatalog.CreditThenInvoiceSettlementMode,
	); err != nil {
		return nil, fmt.Errorf("credits only usage accrued correction: %w", err)
	}

	if input.Charge.Intent.GetCurrency().IsCustom() {
		return nil, fmt.Errorf("usage based charge with custom currency: %w", meta.ErrCustomCurrencyNotSupported)
	}

	currency := intent.GetCurrency()

	if err := input.ValidateWith(currency); err != nil {
		return nil, err
	}

	return h.collector.CorrectCollectedAccrued(ctx, collector.CorrectCollectedAccruedInput{
		Namespace:                    input.Charge.Namespace,
		ChargeID:                     input.Charge.ID,
		CustomerID:                   intent.GetCustomerID(),
		Annotations:                  chargeAnnotationsForUsageBasedCharge(input.Charge),
		AllocateAt:                   input.BookedAt,
		Corrections:                  input.Corrections,
		LineageSegmentsByRealization: input.LineageSegmentsByRealization,
	})
}
