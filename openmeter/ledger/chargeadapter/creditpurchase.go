package chargeadapter

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	chargecreditpurchase "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

// creditPurchaseHandler maps credit purchase lifecycle events to ledger transaction templates.
type creditPurchaseHandler struct {
	ledger          ledger.Ledger
	balanceQuerier  ledger.BalanceQuerier
	accountResolver ledger.AccountResolver
	accountCatalog  ledger.AccountCatalog

	breakage           breakage.Service
	transactionManager transaction.Creator
}

var _ chargecreditpurchase.Handler = (*creditPurchaseHandler)(nil)

func NewCreditPurchaseHandler(
	ledger ledger.Ledger,
	balanceQuerier ledger.BalanceQuerier,
	accountResolver ledger.AccountResolver,
	accountCatalog ledger.AccountCatalog,
	breakageService breakage.Service,
	transactionManager transaction.Creator,
) (chargecreditpurchase.Handler, error) {
	if breakageService == nil {
		breakageService = breakage.NewNoopService()
	}
	if transactionManager == nil {
		return nil, fmt.Errorf("transaction manager is required")
	}

	return &creditPurchaseHandler{
		ledger:             ledger,
		balanceQuerier:     balanceQuerier,
		accountResolver:    accountResolver,
		accountCatalog:     accountCatalog,
		breakage:           breakageService,
		transactionManager: transactionManager,
	}, nil
}

func (h *creditPurchaseHandler) OnPromotionalCreditPurchase(ctx context.Context, charge chargecreditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	return h.issueCreditPurchase(ctx, charge)
}

func (h *creditPurchaseHandler) OnCreditPurchaseInitiated(ctx context.Context, charge chargecreditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	return h.issueCreditPurchase(ctx, charge)
}

func (h *creditPurchaseHandler) OnCreditPurchasePaymentAuthorized(ctx context.Context, input chargecreditpurchase.PaymentEventInput) (ledgertransaction.GroupReference, error) {
	if err := input.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}
	charge := input.Charge

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
			At:        input.EventAt,
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

func (h *creditPurchaseHandler) OnCreditPurchasePaymentSettled(ctx context.Context, input chargecreditpurchase.PaymentEventInput) (ledgertransaction.GroupReference, error) {
	if err := input.Validate(); err != nil {
		return ledgertransaction.GroupReference{}, err
	}
	charge := input.Charge

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
			At:        input.EventAt,
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
	return transaction.Run(ctx, h.transactionManager, func(ctx context.Context) (ledgertransaction.GroupReference, error) {
		return h.issueCreditPurchaseGroup(ctx, charge)
	})
}

func (h *creditPurchaseHandler) issueCreditPurchaseGroup(ctx context.Context, charge chargecreditpurchase.Charge) (ledgertransaction.GroupReference, error) {
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

	advanceAttributions, err := h.advanceAttributions(ctx, customerID, charge.Intent.Currency, charge.Intent.CreditAmount)
	if err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("get advance attributions: %w", err)
	}

	advanceAttributionAmount := alpacadecimal.Zero
	for _, attribution := range advanceAttributions {
		advanceAttributionAmount = advanceAttributionAmount.Add(attribution.advanceAmount)
	}

	issuableAmount := charge.Intent.CreditAmount.Sub(advanceAttributionAmount)
	if issuableAmount.IsNegative() {
		issuableAmount = alpacadecimal.Zero
	}

	var templates []transactions.TransactionTemplate

	for _, attribution := range advanceAttributions {
		templates = append(templates, transactions.AttributeCustomerAdvanceReceivableCostBasisTemplate{
			At:        charge.CreatedAt,
			Amount:    attribution.advanceAmount,
			Currency:  charge.Intent.Currency,
			CostBasis: &costBasis,
		})

		if attribution.accruedAmount.IsPositive() {
			templates = append(templates, transactions.TranslateCustomerAccruedCostBasisTemplate{
				At:            charge.CreatedAt,
				Amount:        attribution.accruedAmount,
				Currency:      charge.Intent.Currency,
				TaxCode:       attribution.taxCode,
				TaxBehavior:   attribution.taxBehavior,
				FromCostBasis: nil,
				ToCostBasis:   &costBasis,
			})
		}
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

	switch charge.Intent.Settlement.Type() {
	case chargecreditpurchase.SettlementTypePromotional:
		// Promotional grants settle immediately through wash so the credited FBO balance
		// does not leave an unsettled receivable behind.
		templates = append(templates,
			transactions.AuthorizeCustomerReceivablePaymentTemplate{
				At:        charge.CreatedAt,
				Amount:    charge.Intent.CreditAmount,
				Currency:  charge.Intent.Currency,
				CostBasis: &costBasis,
			},
			transactions.SettleCustomerReceivableFromPaymentTemplate{
				At:        charge.CreatedAt,
				Amount:    charge.Intent.CreditAmount,
				Currency:  charge.Intent.Currency,
				CostBasis: &costBasis,
			},
		)
	case chargecreditpurchase.SettlementTypeExternal, chargecreditpurchase.SettlementTypeInvoice:
		// Deferred settlement modes are handled by later lifecycle events.
	default:
		return ledgertransaction.GroupReference{}, fmt.Errorf("unsupported settlement type: %s", charge.Intent.Settlement.Type())
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

	var pendingBreakage []breakage.PendingRecord
	if charge.Intent.ExpiresAt != nil {
		breakageInputs, pending, err := h.breakage.PlanIssuance(ctx, breakage.PlanIssuanceInput{
			CustomerID:             customerID,
			Amount:                 charge.Intent.CreditAmount,
			ImmediateReleaseAmount: advanceAttributionAmount,
			Currency:               charge.Intent.Currency,
			CostBasis:              &costBasis,
			CreditPriority:         charge.Intent.Priority,
			ExpiresAt:              *charge.Intent.ExpiresAt,
		})
		if err != nil {
			return ledgertransaction.GroupReference{}, fmt.Errorf("resolve breakage plan: %w", err)
		}

		inputs = append(inputs, breakageInputs...)
		pendingBreakage = append(pendingBreakage, pending...)
	}

	if len(inputs) == 0 {
		return ledgertransaction.GroupReference{}, nil
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

	if err := h.breakage.PersistCommittedRecords(ctx, pendingBreakage, transactionGroup); err != nil {
		return ledgertransaction.GroupReference{}, fmt.Errorf("persist breakage records: %w", err)
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

type advanceAttribution struct {
	taxCode       *string
	taxBehavior   *ledger.TaxBehavior
	advanceAmount alpacadecimal.Decimal
	accruedAmount alpacadecimal.Decimal
}

type unattributedAccruedBalance struct {
	taxCode     *string
	taxBehavior *ledger.TaxBehavior
	amount      alpacadecimal.Decimal
}

type taxDimensionKey struct {
	taxCode     string
	taxBehavior string
}

func (h *creditPurchaseHandler) advanceAttributions(ctx context.Context, customerID customer.CustomerID, currency currencyx.Code, amount alpacadecimal.Decimal) ([]advanceAttribution, error) {
	customerAccounts, err := h.accountResolver.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer accounts: %w", err)
	}

	openStatus := ledger.TransactionAuthorizationStatusOpen
	advanceReceivables, err := h.accountCatalog.ListSubAccounts(ctx, ledger.ListSubAccountsInput{
		Namespace: customerAccounts.ReceivableAccount.ID().Namespace,
		AccountID: customerAccounts.ReceivableAccount.ID().ID,
		Route: ledger.RouteFilter{
			Currency:                       currency,
			CostBasis:                      mo.Some[*alpacadecimal.Decimal](nil),
			TransactionAuthorizationStatus: &openStatus,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("list advance receivable sub-accounts: %w", err)
	}

	unattributedAccrued, err := h.unattributedAccruedBalances(ctx, customerAccounts.AccruedAccount, currency)
	if err != nil {
		return nil, err
	}

	remaining := amount
	attributions := make([]advanceAttribution, 0, len(advanceReceivables))
	for _, advanceReceivable := range advanceReceivables {
		if !remaining.IsPositive() {
			break
		}

		balance, err := settledBalanceForSubAccount(ctx, h.balanceQuerier, advanceReceivable)
		if err != nil {
			return nil, err
		}
		if !balance.IsNegative() {
			continue
		}

		attributed := balance.Neg()
		if attributed.GreaterThan(remaining) {
			attributed = remaining
		}

		accruedRemaining := attributed
		for i := range unattributedAccrued {
			if !accruedRemaining.IsPositive() {
				break
			}

			accrued := unattributedAccrued[i].amount
			if !accrued.IsPositive() {
				continue
			}
			if accrued.GreaterThan(accruedRemaining) {
				accrued = accruedRemaining
			}

			attributions = append(attributions, advanceAttribution{
				taxCode:       unattributedAccrued[i].taxCode,
				taxBehavior:   unattributedAccrued[i].taxBehavior,
				advanceAmount: accrued,
				accruedAmount: accrued,
			})
			unattributedAccrued[i].amount = unattributedAccrued[i].amount.Sub(accrued)
			accruedRemaining = accruedRemaining.Sub(accrued)
		}

		if accruedRemaining.IsPositive() {
			attributions = append(attributions, advanceAttribution{
				advanceAmount: accruedRemaining,
			})
		}
		remaining = remaining.Sub(attributed)
	}

	return attributions, nil
}

func (h *creditPurchaseHandler) unattributedAccruedBalances(ctx context.Context, accruedAccount ledger.CustomerAccruedAccount, currency currencyx.Code) ([]unattributedAccruedBalance, error) {
	subAccounts, err := h.accountCatalog.ListSubAccounts(ctx, ledger.ListSubAccountsInput{
		Namespace: accruedAccount.ID().Namespace,
		AccountID: accruedAccount.ID().ID,
		Route: ledger.RouteFilter{
			Currency:  currency,
			CostBasis: mo.Some[*alpacadecimal.Decimal](nil),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("list unattributed accrued sub-accounts: %w", err)
	}

	balancesByKey := make(map[taxDimensionKey]unattributedAccruedBalance, len(subAccounts))
	keys := make([]taxDimensionKey, 0, len(subAccounts))
	for _, subAccount := range subAccounts {
		balance, err := settledBalanceForSubAccount(ctx, h.balanceQuerier, subAccount)
		if err != nil {
			return nil, err
		}
		if !balance.IsPositive() {
			continue
		}

		route := subAccount.Route()
		key := taxDimensionRouteKey(route)
		if _, ok := balancesByKey[key]; !ok {
			keys = append(keys, key)
			balancesByKey[key] = unattributedAccruedBalance{
				taxCode:     route.TaxCode,
				taxBehavior: route.TaxBehavior,
			}
		}

		current := balancesByKey[key]
		current.amount = current.amount.Add(balance)
		balancesByKey[key] = current
	}

	return lo.Map(keys, func(key taxDimensionKey, _ int) unattributedAccruedBalance {
		return balancesByKey[key]
	}), nil
}

func taxDimensionRouteKey(route ledger.Route) taxDimensionKey {
	return taxDimensionKey{
		taxCode:     lo.FromPtrOr(route.TaxCode, "null"),
		taxBehavior: string(lo.FromPtrOr(route.TaxBehavior, "null")),
	}
}
