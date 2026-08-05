package chargeadapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
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
			Currency:      intent.GetCurrency().Reference(),
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

	if err := validateSettlementMode(
		intent.GetSettlementMode(),
		productcatalog.CreditThenInvoiceSettlementMode,
	); err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("payment authorized: %w", err)
	}

	invoiceCurrency, err := input.Charge.GetInvoiceCurrency()
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("get invoice currency: %w", err)
	}

	costBasis, err := resolveInvoiceCostBasis(input.Charge)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("payment authorized: %w", err)
	}

	customerID := customer.CustomerID{
		Namespace: input.Charge.Namespace,
		ID:        intent.GetCustomerID(),
	}
	annotations := chargeAnnotationsForUsageBasedCharge(input.Charge)
	paymentIdentity := invoicePaymentIdentity(input.Charge.ID, intent.GetCurrency())

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.deps,
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  input.Charge.Namespace,
		},
		transactions.AuthorizeCustomerReceivablePaymentTemplate{
			At:             input.EventAt,
			Amount:         input.FiatAmount,
			Currency:       currencies.NewCurrencyReference(currencyx.Code(invoiceCurrency)),
			CostBasis:      costBasis,
			SourceChargeID: paymentIdentity.SourceChargeID,
			SpendChargeID:  paymentIdentity.SpendChargeID,
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

// OnCustomCurrencyOverageAccrued books uncovered custom-currency overage as a
// credit purchase immediately consumed by the same charge: it purchases the
// uncovered custom amount at the charge's persisted cost basis, consumes it
// through the identical route so it never becomes spendable customer balance,
// and converts the resulting custom-currency receivable into the
// already-agreed, rounded fiat receivable the invoice will collect. This
// keeps native amount, cost basis, and fiat provenance on the accrued leg
// while the invoice and payment lifecycle stay entirely in fiat.
func (h *usageBasedHandler) OnCustomCurrencyOverageAccrued(ctx context.Context, input usagebased.OnCustomCurrencyOverageAccruedInput) (usagebased.OnCustomCurrencyOverageAccruedResult, error) {
	if err := input.Validate(); err != nil {
		return usagebased.OnCustomCurrencyOverageAccruedResult{}, err
	}

	fiatOverage, err := input.Charge.ConvertCustomCurrencyOverageToFiat(input.Run.Totals)
	if err != nil {
		return usagebased.OnCustomCurrencyOverageAccruedResult{}, fmt.Errorf("convert custom currency overage to fiat: %w", err)
	}

	costBasis, err := input.GetCostBasis()
	if err != nil {
		return usagebased.OnCustomCurrencyOverageAccruedResult{}, fmt.Errorf("get cost basis: %w", err)
	}

	intent := input.Charge.Intent
	taxConfig := intent.GetTaxConfig()
	customCurrency := input.CustomCurrency().Reference()
	fiatCode := currencyx.Code(fiatOverage.Currency.GetFiatCode())
	customAmount := input.GetCustomCurrencyAmountAccrued()

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
		// Purchase the uncovered custom amount at the charge's persisted cost basis.
		transactions.IssueCustomerReceivableTemplate{
			At:                input.Run.ServicePeriodTo,
			Amount:            customAmount,
			Currency:          customCurrency,
			CostBasisCurrency: &fiatCode,
			CostBasis:         &costBasis,
			SourceChargeID:    &input.Charge.ID,
		},
		// Immediately consume the purchase through the same charge, so it never
		// becomes spendable customer balance.
		transactions.TransferCustomerFBOAdvanceToAccruedTemplate{
			At:                input.Run.ServicePeriodTo,
			Amount:            customAmount,
			Currency:          customCurrency,
			TaxCode:           lo.ToPtr(taxConfig.TaxCodeID),
			TaxBehavior:       (*ledger.TaxBehavior)(taxConfig.Behavior),
			CostBasisCurrency: &fiatCode,
			CostBasis:         &costBasis,
			SourceChargeID:    &input.Charge.ID,
			SpendChargeID:     &input.Charge.ID,
		},
		// Convert the purchase-backed custom receivable into the already-agreed,
		// rounded fiat receivable the invoice will collect.
		transactions.ConvertCurrencyTemplate{
			At:             input.Run.ServicePeriodTo,
			SourceAmount:   fiatOverage.Amount,
			TargetAmount:   customAmount,
			CostBasis:      costBasis,
			SourceCurrency: currencies.NewCurrencyReference(fiatCode),
			TargetCurrency: customCurrency,
		},
	)
	if err != nil {
		return usagebased.OnCustomCurrencyOverageAccruedResult{}, fmt.Errorf("resolve transactions: %w", err)
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
		return usagebased.OnCustomCurrencyOverageAccruedResult{}, fmt.Errorf("commit ledger transaction group: %w", err)
	}

	return usagebased.OnCustomCurrencyOverageAccruedResult{
		TransactionGroup: ledgertransaction.GroupReference{
			TransactionGroupID: transactionGroup.ID().ID,
		},
		TotalFiatAmount: fiatOverage.Amount,
	}, nil
}

func (h *usageBasedHandler) OnCustomCurrencyOverageAccruedCorrection(ctx context.Context, input usagebased.OnCustomCurrencyOverageAccruedCorrectionInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return fmt.Errorf("implement OnCustomCurrencyOverageAccruedCorrection: %w", meta.ErrCustomCurrencyNotSupported)
}

func (h *usageBasedHandler) OnAllocateFiatOverageCredits(ctx context.Context, input usagebased.AllocateFiatOverageCreditsInput) (creditrealization.CreateAllocationInputs, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// TODO[implement]: Allocate settlement-fiat credits against the custom-currency overage.
	return nil, fmt.Errorf("implement OnAllocateFiatOverageCredits: %w", meta.ErrCustomCurrencyNotSupported)
}

func (h *usageBasedHandler) OnCorrectFiatOverageCreditAllocations(ctx context.Context, input usagebased.CorrectFiatOverageCreditAllocationsInput) (creditrealization.CreateCorrectionInputs, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// TODO[implement]: Correct settlement-fiat allocations for the custom-currency overage.
	return nil, fmt.Errorf("implement OnCorrectFiatOverageCreditAllocations: %w", meta.ErrCustomCurrencyNotSupported)
}

func (h *usageBasedHandler) OnPaymentSettled(ctx context.Context, input usagebased.OnPaymentSettledInput) (ledgertransaction.GroupReference, error) {
	if err := input.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	intent := input.Charge.Intent

	if err := validateSettlementMode(
		intent.GetSettlementMode(),
		productcatalog.CreditThenInvoiceSettlementMode,
	); err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("payment settled: %w", err)
	}

	invoiceCurrency, err := input.Charge.GetInvoiceCurrency()
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("get invoice currency: %w", err)
	}

	costBasis, err := resolveInvoiceCostBasis(input.Charge)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("payment settled: %w", err)
	}

	customerID := customer.CustomerID{
		Namespace: input.Charge.Namespace,
		ID:        intent.GetCustomerID(),
	}
	annotations := chargeAnnotationsForUsageBasedCharge(input.Charge)
	paymentIdentity := invoicePaymentIdentity(input.Charge.ID, intent.GetCurrency())

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.deps,
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  input.Charge.Namespace,
		},
		transactions.SettleCustomerReceivableFromPaymentTemplate{
			At:             input.EventAt,
			Amount:         input.FiatAmount,
			Currency:       currencies.NewCurrencyReference(currencyx.Code(invoiceCurrency)),
			CostBasis:      costBasis,
			SourceChargeID: paymentIdentity.SourceChargeID,
			SpendChargeID:  paymentIdentity.SpendChargeID,
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
		Currency:          intent.GetCurrency().Reference(),
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
