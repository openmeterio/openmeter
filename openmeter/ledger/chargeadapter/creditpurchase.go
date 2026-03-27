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
	"github.com/openmeterio/openmeter/pkg/models"
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
	if err := charge.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	if charge.Intent.CreditAmount.IsZero() {
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

	costBasis := alpacadecimal.Zero
	advanceOutstanding, err := h.outstandingAdvanceBalance(ctx, customerID, charge.Intent.Currency)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("get outstanding advance balance: %w", err)
	}

	issuableAmount := charge.Intent.CreditAmount.Sub(advanceOutstanding)
	if issuableAmount.IsNegative() {
		issuableAmount = alpacadecimal.Zero
	}
	if issuableAmount.IsZero() {
		return ledgertransaction.GroupReference{}, nil
	}

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.resolverDependencies(),
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  charge.Namespace,
		},
		transactions.IssueCustomerReceivableTemplate{
			At:        charge.CreatedAt,
			Amount:    charge.Intent.CreditAmount,
			Currency:  charge.Intent.Currency,
			CostBasis: &costBasis,
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

func (h *creditPurchaseHandler) OnCreditPurchaseInitiated(ctx context.Context, charge chargecreditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	if err := charge.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	if charge.Intent.CreditAmount.IsZero() {
		return ledgertransaction.GroupReference{}, nil
	}

	externalSettlement, err := charge.Intent.Settlement.AsExternalSettlement()
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("external settlement: %w", err)
	}

	customerID := customer.CustomerID{
		Namespace: charge.Namespace,
		ID:        charge.Intent.CustomerID,
	}
	annotations := ledger.ChargeAnnotations(models.NamespacedID{
		Namespace: charge.Namespace,
		ID:        charge.ID,
	})

	advanceOutstanding, err := h.outstandingAdvanceBalance(ctx, customerID, charge.Intent.Currency)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("get outstanding advance balance: %w", err)
	}

	issuableAmount := charge.Intent.CreditAmount.Sub(advanceOutstanding)
	if issuableAmount.IsNegative() {
		issuableAmount = alpacadecimal.Zero
	}
	if issuableAmount.IsZero() {
		return ledgertransaction.GroupReference{}, nil
	}

	inputs, err := transactions.ResolveTransactions(
		ctx,
		h.resolverDependencies(),
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  charge.Namespace,
		},
		transactions.IssueCustomerReceivableTemplate{
			At:        charge.CreatedAt,
			Amount:    issuableAmount,
			Currency:  charge.Intent.Currency,
			CostBasis: &externalSettlement.CostBasis,
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

func (h *creditPurchaseHandler) OnCreditPurchasePaymentAuthorized(ctx context.Context, charge chargecreditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	if err := charge.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	externalSettlement, err := charge.Intent.Settlement.AsExternalSettlement()
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("external settlement: %w", err)
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
		transactions.FundCustomerReceivableTemplate{
			At:        charge.CreatedAt,
			Amount:    charge.Intent.CreditAmount,
			Currency:  charge.Intent.Currency,
			CostBasis: &externalSettlement.CostBasis,
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

func (h *creditPurchaseHandler) OnCreditPurchasePaymentSettled(ctx context.Context, charge chargecreditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	if err := charge.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	externalSettlement, err := charge.Intent.Settlement.AsExternalSettlement()
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("external settlement: %w", err)
	}

	customerID := customer.CustomerID{
		Namespace: charge.Namespace,
		ID:        charge.Intent.CustomerID,
	}
	annotations := ledger.ChargeAnnotations(models.NamespacedID{
		Namespace: charge.Namespace,
		ID:        charge.ID,
	})

	advanceOutstanding, err := h.outstandingAdvanceBalance(ctx, customerID, charge.Intent.Currency)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("get outstanding advance balance: %w", err)
	}

	unattributedAccrued, err := h.unattributedAccruedBalance(ctx, customerID, charge.Intent.Currency)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("get unattributed accrued balance: %w", err)
	}

	backableAmount := charge.Intent.CreditAmount
	if backableAmount.GreaterThan(advanceOutstanding) {
		backableAmount = advanceOutstanding
	}
	if backableAmount.GreaterThan(unattributedAccrued) {
		backableAmount = unattributedAccrued
	}

	templates := []transactions.Resolver{
		transactions.SettleCustomerReceivablePaymentTemplate{
			At:        charge.CreatedAt,
			Amount:    charge.Intent.CreditAmount,
			Currency:  charge.Intent.Currency,
			CostBasis: &externalSettlement.CostBasis,
		},
	}
	if backableAmount.IsPositive() {
		templates = append(templates,
			transactions.FundCustomerAdvanceReceivableTemplate{
				At:        charge.CreatedAt,
				Amount:    backableAmount,
				Currency:  charge.Intent.Currency,
				CostBasis: &externalSettlement.CostBasis,
			},
			transactions.TranslateCustomerAccruedCostBasisTemplate{
				At:            charge.CreatedAt,
				Amount:        backableAmount,
				Currency:      charge.Intent.Currency,
				FromCostBasis: nil,
				ToCostBasis:   &externalSettlement.CostBasis,
			},
		)
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
