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
	"github.com/openmeterio/openmeter/pkg/models"
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
func (h *flatFeeHandler) OnAssignedToInvoice(ctx context.Context, input flatfee.OnAssignedToInvoiceInput) ([]creditrealization.CreateInput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	if input.PreTaxTotalAmount.IsZero() {
		return nil, nil
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
		transactions.TransferCustomerFBOToAccruedTemplate{
			At:       input.Charge.Intent.InvoiceAt,
			Amount:   input.PreTaxTotalAmount,
			Currency: input.Charge.Intent.Currency,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("resolve transactions: %w", err)
	}
	if len(inputs) == 0 {
		return nil, nil
	}

	transactionGroup, err := h.ledger.CommitGroup(ctx, transactions.GroupInputs(
		input.Charge.Namespace,
		annotations,
		inputs...,
	))
	if err != nil {
		return nil, fmt.Errorf("commit ledger transaction group: %w", err)
	}

	totalCollected := sumPositiveEntryAmounts(inputs...)

	return []creditrealization.CreateInput{
		{
			ServicePeriod: input.ServicePeriod,
			Amount:        totalCollected,
			LedgerTransaction: ledgertransaction.GroupReference{
				TransactionGroupID: transactionGroup.ID().ID,
			},
		},
	}, nil
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
			At:       input.Charge.Intent.InvoiceAt,
			Amount:   amount,
			Currency: input.Charge.Intent.Currency,
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
func (h *flatFeeHandler) OnCreditsOnlyUsageAccrued(ctx context.Context, input flatfee.OnCreditsOnlyUsageAccruedInput) ([]creditrealization.CreateInput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	if input.AmountToAllocate.IsZero() {
		return nil, nil
	}

	return nil, fmt.Errorf("on credits only usage accrued is not implemented")
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

	// The receivable portion needs wash → receivable replenishment.
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

	customerAccounts, err := h.accountResolver.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("get customer accounts: %w", err)
	}

	// Recognize revenue: move from accrued to earnings.
	accruedSubAccount, err := customerAccounts.AccruedAccount.GetSubAccountForRoute(ctx, ledger.CustomerAccruedRouteParams{
		Currency: charge.Intent.Currency,
	})
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("get accrued sub-account: %w", err)
	}

	accruedBalance, err := settledBalanceForSubAccount(ctx, accruedSubAccount)
	if err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	recognitionAmount := totalRecognition
	if recognitionAmount.GreaterThan(accruedBalance) {
		recognitionAmount = accruedBalance
	}

	var templates []transactions.Resolver
	if receivableReplenishment.IsPositive() {
		templates = append(templates, transactions.FundCustomerReceivableTemplate{
			At:       charge.Intent.InvoiceAt,
			Amount:   receivableReplenishment,
			Currency: charge.Intent.Currency,
		})
	}
	if recognitionAmount.IsPositive() {
		templates = append(templates, transactions.RecognizeEarningsFromAccruedTemplate{
			At:       charge.Intent.InvoiceAt,
			Amount:   recognitionAmount,
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

// OnFlatFeePaymentSettled is intentionally unimplemented for now.
// Later this may be the point where revenue recognition happens instead of authorization.
func (h *flatFeeHandler) OnPaymentSettled(_ context.Context, _ flatfee.Charge) (ledgertransaction.GroupReference, error) {
	return ledgertransaction.GroupReference{}, fmt.Errorf("flat fee payment settlement is not yet implemented")
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

func sumPositiveEntryAmounts(inputs ...ledger.TransactionInput) alpacadecimal.Decimal {
	total := alpacadecimal.Zero
	for _, input := range inputs {
		if input == nil {
			continue
		}
		for _, entry := range input.EntryInputs() {
			if entry.Amount().IsPositive() {
				total = total.Add(entry.Amount())
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
