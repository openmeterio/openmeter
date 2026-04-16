package chargeadapter

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	chargecreditpurchase "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// creditPurchaseHandler maps credit purchase lifecycle events to ledger transaction templates.
type creditPurchaseHandler struct {
	ledger          ledger.Ledger
	accountResolver ledger.AccountResolver
	accountService  ledgeraccount.Service
}

var _ chargecreditpurchase.Handler = (*creditPurchaseHandler)(nil)

func NewCreditPurchaseHandler(
	ledger ledger.Ledger,
	accountResolver ledger.AccountResolver,
	accountService ledgeraccount.Service,
) chargecreditpurchase.Handler {
	return &creditPurchaseHandler{
		ledger:          ledger,
		accountResolver: accountResolver,
		accountService:  accountService,
	}
}

func (h *creditPurchaseHandler) OnPromotionalCreditPurchase(ctx context.Context, charge chargecreditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	return h.issueCreditPurchase(ctx, charge)
}

func (h *creditPurchaseHandler) OnCreditPurchaseInitiated(ctx context.Context, charge chargecreditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	return h.issueCreditPurchase(ctx, charge)
}

func (h *creditPurchaseHandler) OnCreditPurchasePaymentAuthorized(ctx context.Context, charge chargecreditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	if err := charge.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	costBasis, err := charge.Intent.Settlement.GetCostBasis()
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("get cost basis: %w", err)
	}

	customerID := customer.CustomerID{
		Namespace: charge.Namespace,
		ID:        charge.Intent.CustomerID,
	}
	annotations := chargeAnnotationsForCreditPurchaseCharge(charge)

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.resolverDependencies(),
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  charge.Namespace,
		},
		transactions.FundCustomerReceivableTemplate{
			At:        charge.CreatedAt,
			Amount:    charge.Intent.CreditAmount,
			Currency:  charge.Intent.Currency,
			CostBasis: &costBasis,
		},
	)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("resolve transactions: %w", err)
	}

	for i, input := range inputs {
		if input != nil {
			inputs[i] = transactions.WithAnnotations(input, annotations)
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

func (h *creditPurchaseHandler) OnCreditPurchasePaymentSettled(ctx context.Context, charge chargecreditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	if err := charge.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	costBasis, err := charge.Intent.Settlement.GetCostBasis()
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("get cost basis: %w", err)
	}

	customerID := customer.CustomerID{
		Namespace: charge.Namespace,
		ID:        charge.Intent.CustomerID,
	}
	annotations := chargeAnnotationsForCreditPurchaseCharge(charge)

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.resolverDependencies(),
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  charge.Namespace,
		},
		transactions.SettleCustomerReceivablePaymentTemplate{
			At:        charge.CreatedAt,
			Amount:    charge.Intent.CreditAmount,
			Currency:  charge.Intent.Currency,
			CostBasis: &costBasis,
		},
	)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("resolve transactions: %w", err)
	}

	for i, input := range inputs {
		if input != nil {
			inputs[i] = transactions.WithAnnotations(input, annotations)
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

// issueCreditPurchase is the shared logic for issuing credits (both promotional and externally settled).
// It attributes outstanding advance receivables and unattributed accrued balances to the given cost basis,
// then issues new receivables for any remaining amount.
func (h *creditPurchaseHandler) issueCreditPurchase(ctx context.Context, charge chargecreditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	if err := charge.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	if charge.Intent.CreditAmount.IsZero() {
		return ledgertransaction.GroupReference{}, nil
	}

	costBasis, err := charge.Intent.Settlement.GetCostBasis()
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("get cost basis: %w", err)
	}

	customerID := customer.CustomerID{
		Namespace: charge.Namespace,
		ID:        charge.Intent.CustomerID,
	}
	annotations := chargeAnnotationsForCreditPurchaseCharge(charge)

	advanceOutstanding, err := h.outstandingAdvanceBalance(ctx, customerID, charge.Intent.Currency)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("get outstanding advance balance: %w", err)
	}

	unattributedAccrued, err := h.unattributedAccruedBalance(ctx, customerID, charge.Intent.Currency)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("get unattributed accrued balance: %w", err)
	}

	advanceAttributionAmount := charge.Intent.CreditAmount
	if advanceAttributionAmount.GreaterThan(advanceOutstanding) {
		advanceAttributionAmount = advanceOutstanding
	}

	accruedAttributionAmount := advanceAttributionAmount
	if accruedAttributionAmount.GreaterThan(unattributedAccrued) {
		accruedAttributionAmount = unattributedAccrued
	}

	issuableAmount := charge.Intent.CreditAmount.Sub(advanceAttributionAmount)
	if issuableAmount.IsNegative() {
		issuableAmount = alpacadecimal.Zero
	}

	var templates []transactions.TransactionTemplate

	if advanceAttributionAmount.IsPositive() {
		templates = append(templates, transactions.AttributeCustomerAdvanceReceivableCostBasisTemplate{
			At:        charge.CreatedAt,
			Amount:    advanceAttributionAmount,
			Currency:  charge.Intent.Currency,
			CostBasis: &costBasis,
		})
	}

	if accruedAttributionAmount.IsPositive() {
		templates = append(templates, transactions.TranslateCustomerAccruedCostBasisTemplate{
			At:            charge.CreatedAt,
			Amount:        accruedAttributionAmount,
			Currency:      charge.Intent.Currency,
			FromCostBasis: nil,
			ToCostBasis:   &costBasis,
		})
	}

	if issuableAmount.IsPositive() {
		templates = append(templates, transactions.IssueCustomerReceivableTemplate{
			At:             charge.CreatedAt,
			Amount:         issuableAmount,
			Currency:       charge.Intent.Currency,
			CostBasis:      &costBasis,
			CreditPriority: charge.Intent.Priority,
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

	for i, input := range inputs {
		if input != nil {
			inputs[i] = transactions.WithAnnotations(input, annotations)
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

func (h *creditPurchaseHandler) resolverDependencies() transactions.ResolverDependencies {
	return transactions.ResolverDependencies{
		AccountService:    h.accountResolver,
		SubAccountService: h.accountService,
	}
}

func (h *creditPurchaseHandler) outstandingAdvanceBalance(ctx context.Context, customerID customer.CustomerID, currency currencyx.Code) (alpacadecimal.Decimal, error) {
	customerAccounts, err := h.accountResolver.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return alpacadecimal.Decimal{}, fmt.Errorf("get customer accounts: %w", err)
	}

	advanceReceivable, err := customerAccounts.ReceivableAccount.GetSubAccountForRoute(ctx, ledger.CustomerReceivableRouteParams{
		Currency:                       currency,
		CostBasis:                      nil,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	if err != nil {
		return alpacadecimal.Decimal{}, fmt.Errorf("get advance receivable sub-account: %w", err)
	}

	balance, err := settledBalanceForSubAccount(ctx, advanceReceivable)
	if err != nil {
		return alpacadecimal.Decimal{}, err
	}

	if balance.IsNegative() {
		return balance.Neg(), nil
	}

	return alpacadecimal.Zero, nil
}

func (h *creditPurchaseHandler) unattributedAccruedBalance(ctx context.Context, customerID customer.CustomerID, currency currencyx.Code) (alpacadecimal.Decimal, error) {
	customerAccounts, err := h.accountResolver.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return alpacadecimal.Decimal{}, fmt.Errorf("get customer accounts: %w", err)
	}

	unknownAccrued, err := customerAccounts.AccruedAccount.GetSubAccountForRoute(ctx, ledger.CustomerAccruedRouteParams{
		Currency:  currency,
		CostBasis: nil,
	})
	if err != nil {
		return alpacadecimal.Decimal{}, fmt.Errorf("get unattributed accrued sub-account: %w", err)
	}

	balance, err := settledBalanceForSubAccount(ctx, unknownAccrued)
	if err != nil {
		return alpacadecimal.Decimal{}, err
	}

	if balance.IsPositive() {
		return balance, nil
	}

	return alpacadecimal.Zero, nil
}
