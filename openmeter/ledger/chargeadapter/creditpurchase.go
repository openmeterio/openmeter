package chargeadapter

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	chargecreditpurchase "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// creditPurchaseHandler maps credit purchase lifecycle events to ledger transaction templates.
type creditPurchaseHandler struct {
	ledger          ledger.Ledger
	balanceQuerier  ledger.BalanceQuerier
	accountResolver ledger.AccountResolver
	accountCatalog  ledger.AccountCatalog
}

var _ chargecreditpurchase.Handler = (*creditPurchaseHandler)(nil)

func NewCreditPurchaseHandler(
	ledger ledger.Ledger,
	balanceQuerier ledger.BalanceQuerier,
	accountResolver ledger.AccountResolver,
	accountCatalog ledger.AccountCatalog,
) chargecreditpurchase.Handler {
	return &creditPurchaseHandler{
		ledger:          ledger,
		balanceQuerier:  balanceQuerier,
		accountResolver: accountResolver,
		accountCatalog:  accountCatalog,
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
		transactions.AuthorizeCustomerReceivablePaymentTemplate{
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
		transactions.SettleCustomerReceivableFromPaymentTemplate{
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

	var taxCodeID *string
	if charge.Intent.TaxConfig != nil {
		taxCodeID = charge.Intent.TaxConfig.TaxCodeID
	}

	var templates []transactions.TransactionTemplate

	if advanceAttributionAmount.IsPositive() {
		templates = append(templates, transactions.AttributeCustomerAdvanceReceivableCostBasisTemplate{
			At:        charge.CreatedAt,
			Amount:    advanceAttributionAmount,
			Currency:  charge.Intent.Currency,
			TaxCode:   taxCodeID,
			CostBasis: &costBasis,
		})
	}

	if accruedAttributionAmount.IsPositive() {
		templates = append(templates, transactions.TranslateCustomerAccruedCostBasisTemplate{
			At:            charge.CreatedAt,
			Amount:        accruedAttributionAmount,
			Currency:      charge.Intent.Currency,
			TaxCode:       taxCodeID,
			FromCostBasis: nil,
			ToCostBasis:   &costBasis,
		})
	}

	if issuableAmount.IsPositive() {
		tmpl := transactions.IssueCustomerReceivableTemplate{
			At:             charge.CreatedAt,
			Amount:         issuableAmount,
			Currency:       charge.Intent.Currency,
			TaxCode:        taxCodeID,
			CostBasis:      &costBasis,
			CreditPriority: charge.Intent.Priority,
		}
		if charge.Intent.TaxConfig != nil && charge.Intent.TaxConfig.Behavior != nil {
			b := ledger.TaxBehavior(*charge.Intent.TaxConfig.Behavior)
			tmpl.TaxBehavior = &b
		}
		templates = append(templates, tmpl)
	}

	switch charge.Intent.Settlement.Type() {
	case chargecreditpurchase.SettlementTypePromotional:
		// Promotional grants settle immediately through wash so the credited FBO balance
		// does not leave an unsettled receivable behind.
		templates = append(templates,
			transactions.AuthorizeCustomerReceivablePaymentTemplate{
				At:        charge.CreatedAt,
				Amount:    charge.Intent.CreditAmount,
				Currency:  charge.Intent.Currency,
				TaxCode:   taxCodeID,
				CostBasis: &costBasis,
			},
			transactions.SettleCustomerReceivableFromPaymentTemplate{
				At:        charge.CreatedAt,
				Amount:    charge.Intent.CreditAmount,
				Currency:  charge.Intent.Currency,
				TaxCode:   taxCodeID,
				CostBasis: &costBasis,
			},
		)
	case chargecreditpurchase.SettlementTypeExternal, chargecreditpurchase.SettlementTypeInvoice:
		// Deferred settlement modes are handled by later lifecycle events.
	default:
		return ledgertransaction.GroupReference{}, fmt.Errorf("unsupported settlement type: %s", charge.Intent.Settlement.Type())
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
		AccountService: h.accountResolver,
		AccountCatalog: h.accountCatalog,
		BalanceQuerier: h.balanceQuerier,
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

	balance, err := settledBalanceForSubAccount(ctx, h.balanceQuerier, advanceReceivable)
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

	balance, err := settledBalanceForSubAccount(ctx, h.balanceQuerier, unknownAccrued)
	if err != nil {
		return alpacadecimal.Decimal{}, err
	}

	if balance.IsPositive() {
		return balance, nil
	}

	return alpacadecimal.Zero, nil
}
