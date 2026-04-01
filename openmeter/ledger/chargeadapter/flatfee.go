package chargeadapter

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// flatFeeHandler maps charge lifecycle events to ledger transaction templates
type flatFeeHandler struct {
	ledger          ledger.Ledger
	accountResolver ledger.AccountResolver
	accountService  ledgeraccount.Service
}

var _ flatfee.Handler = (*flatFeeHandler)(nil)

func NewFlatFeeHandler(
	ledger ledger.Ledger,
	accountResolver ledger.AccountResolver,
	accountService ledgeraccount.Service,
) flatfee.Handler {
	return &flatFeeHandler{
		ledger:          ledger,
		accountResolver: accountResolver,
		accountService:  accountService,
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

	groupID, inputs, err := h.allocateCreditsToAccrued(ctx, input.Charge, input.PreTaxTotalAmount)
	if err != nil {
		return nil, err
	}
	if groupID == "" {
		return nil, nil
	}

	return creditRealizationsFromCollectedInputs(input.ServicePeriod, groupID, inputs...), nil
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
	annotations := ledger.ChargeAnnotations(models.NamespacedID{
		Namespace: input.Charge.Namespace,
		ID:        input.Charge.ID,
	})

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.resolverDependencies(),
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  input.Charge.Namespace,
		},
		transactions.TransferCustomerReceivableToAccruedTemplate{
			At:        input.Charge.Intent.InvoiceAt,
			Amount:    amount,
			Currency:  input.Charge.Intent.Currency,
			CostBasis: invoiceCostBasis(),
		},
	)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("resolve transactions: %w", err)
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

	groupID, inputs, err := h.allocateCreditsToAccrued(ctx, input.Charge, input.AmountToAllocate)
	if err != nil {
		return nil, err
	}
	if groupID == "" {
		return nil, nil
	}

	return creditRealizationsFromCollectedInputs(input.Charge.Intent.ServicePeriod, groupID, inputs...), nil
}

func (h *flatFeeHandler) OnCreditsOnlyUsageAccruedCorrection(ctx context.Context, input flatfee.CreditsOnlyUsageAccruedCorrectionInput) (creditrealization.CreateCorrectionInputs, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("credits only usage accrued correction is not implemented")
}

// OnFlatFeePaymentAuthorized is the current revenue recognition point.
// It replenishes receivable from wash for the directly-invoiced portion, and
// recognizes revenue by moving from customer_accrued to earnings.
func (h *flatFeeHandler) OnPaymentAuthorized(ctx context.Context, charge flatfee.Charge) (ledgertransaction.GroupReference, error) {
	if err := charge.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	// Compute the total amount to recognize from accrued into earnings.
	// This includes both credit-backed (FBO) and receivable-backed portions.
	totalRecognition := alpacadecimal.NewFromInt(0)
	for _, cr := range charge.State.CreditRealizations {
		totalRecognition = totalRecognition.Add(cr.Amount)
	}

	// The receivable portion needs wash -> receivable replenishment.
	receivableReplenishment := alpacadecimal.NewFromInt(0)
	if charge.State.AccruedUsage != nil {
		receivableReplenishment = charge.State.AccruedUsage.Totals.Total
		totalRecognition = totalRecognition.Add(charge.State.AccruedUsage.Totals.Total)
	}

	if totalRecognition.IsZero() {
		return ledgertransaction.GroupReference{}, nil
	}

	customerID := customer.CustomerID{
		Namespace: charge.Namespace,
		ID:        charge.Intent.CustomerID,
	}
	annotations := ledger.ChargeAnnotations(models.NamespacedID{
		Namespace: charge.Namespace,
		ID:        charge.ID,
	})

	var templates []transactions.Resolver
	if receivableReplenishment.IsPositive() {
		templates = append(templates, transactions.FundCustomerReceivableTemplate{
			At:        charge.Intent.InvoiceAt,
			Amount:    receivableReplenishment,
			Currency:  charge.Intent.Currency,
			CostBasis: invoiceCostBasis(),
		})
	}
	if totalRecognition.IsPositive() {
		templates = append(templates, transactions.RecognizeEarningsFromAttributableAccruedTemplate{
			At:       charge.Intent.InvoiceAt,
			Amount:   totalRecognition,
			Currency: charge.Intent.Currency,
		})
	}

	if len(templates) == 0 {
		return ledgertransaction.GroupReference{}, nil
	}

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.resolverDependencies(),
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  charge.Namespace,
		},
		templates...,
	)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("resolve transactions: %w", err)
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

	if charge.State.AccruedUsage == nil || !charge.State.AccruedUsage.Totals.Total.IsPositive() {
		return ledgertransaction.GroupReference{}, nil
	}

	customerID := customer.CustomerID{
		Namespace: charge.Namespace,
		ID:        charge.Intent.CustomerID,
	}
	annotations := ledger.ChargeAnnotations(models.NamespacedID{
		Namespace: charge.Namespace,
		ID:        charge.ID,
	})

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.resolverDependencies(),
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  charge.Namespace,
		},
		transactions.SettleCustomerReceivablePaymentTemplate{
			At:        charge.Intent.InvoiceAt,
			Amount:    charge.State.AccruedUsage.Totals.Total,
			Currency:  charge.Intent.Currency,
			CostBasis: invoiceCostBasis(),
		},
	)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("resolve transactions: %w", err)
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

func (h *flatFeeHandler) resolverDependencies() transactions.ResolverDependencies {
	return transactions.ResolverDependencies{
		AccountService:    h.accountResolver,
		SubAccountService: h.accountService,
	}
}

func validateSettlementMode(actual productcatalog.SettlementMode, allowed ...productcatalog.SettlementMode) error {
	for _, candidate := range allowed {
		if actual == candidate {
			return nil
		}
	}

	return fmt.Errorf("unsupported settlement mode %q", actual)
}

func (h *flatFeeHandler) allocateCreditsToAccrued(ctx context.Context, charge flatfee.Charge, amount alpacadecimal.Decimal) (string, []ledger.TransactionInput, error) {
	customerID := customer.CustomerID{
		Namespace: charge.Namespace,
		ID:        charge.Intent.CustomerID,
	}
	annotations := ledger.ChargeAnnotations(models.NamespacedID{
		Namespace: charge.Namespace,
		ID:        charge.ID,
	})

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.resolverDependencies(),
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  charge.Namespace,
		},
		transactions.TransferCustomerFBOToAccruedTemplate{
			At:       charge.Intent.InvoiceAt,
			Amount:   amount,
			Currency: charge.Intent.Currency,
		},
	)
	if err != nil {
		return "", nil, fmt.Errorf("resolve transactions: %w", err)
	}

	collectedAmount := sumCollectedFBOAmount(inputs...)
	shortfall := amount.Sub(collectedAmount)
	if charge.Intent.SettlementMode == productcatalog.CreditOnlySettlementMode && shortfall.IsPositive() {
		advanceInputs, err := transactions.ResolveTransactions(
			ctx,
			h.resolverDependencies(),
			transactions.ResolutionScope{
				CustomerID: customerID,
				Namespace:  charge.Namespace,
			},
			transactions.IssueCustomerReceivableTemplate{
				At:       charge.Intent.InvoiceAt,
				Amount:   shortfall,
				Currency: charge.Intent.Currency,
			},
			transactions.TransferCustomerFBOBucketToAccruedTemplate{
				At:       charge.Intent.InvoiceAt,
				Amount:   shortfall,
				Currency: charge.Intent.Currency,
			},
		)
		if err != nil {
			return "", nil, fmt.Errorf("resolve advance transactions: %w", err)
		}

		inputs = append(inputs, advanceInputs...)
	}

	if len(inputs) == 0 {
		return "", nil, nil
	}

	transactionGroup, err := h.ledger.CommitGroup(ctx, transactions.GroupInputs(
		charge.Namespace,
		annotations,
		inputs...,
	))
	if err != nil {
		return "", nil, fmt.Errorf("commit ledger transaction group: %w", err)
	}

	return transactionGroup.ID().ID, inputs, nil
}

func creditRealizationsFromCollectedInputs(servicePeriod timeutil.ClosedPeriod, transactionGroupID string, inputs ...ledger.TransactionInput) creditrealization.CreateAllocationInputs {
	out := make(creditrealization.CreateAllocationInputs, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		for _, entry := range input.EntryInputs() {
			if entry.Amount().IsNegative() && entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO {
				out = append(out, creditrealization.CreateAllocationInput{
					ServicePeriod: servicePeriod,
					Amount:        entry.Amount().Abs(),
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: transactionGroupID,
					},
				})
			}
		}
	}

	return out
}

func invoiceCostBasis() *alpacadecimal.Decimal {
	value := alpacadecimal.NewFromInt(1)
	return &value
}

func sumCollectedFBOAmount(inputs ...ledger.TransactionInput) alpacadecimal.Decimal {
	total := alpacadecimal.Zero
	for _, input := range inputs {
		if input == nil {
			continue
		}
		for _, entry := range input.EntryInputs() {
			if entry.Amount().IsNegative() && entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO {
				total = total.Add(entry.Amount().Abs())
			}
		}
	}

	return total
}

func settledBalanceForSubAccount(ctx context.Context, subAccount ledger.SubAccount) (alpacadecimal.Decimal, error) {
	balance, err := subAccount.GetBalance(ctx)
	if err != nil {
		return alpacadecimal.Decimal{}, fmt.Errorf("get balance for sub-account %s: %w", subAccount.Address().SubAccountID(), err)
	}

	return balance.Settled(), nil
}
