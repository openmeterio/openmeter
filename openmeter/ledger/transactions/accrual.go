package transactions

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// TransferCustomerFBOToAccruedTemplate moves value from prioritized customer FBO
// sub-accounts into the customer's accrued account.
type TransferCustomerFBOToAccruedTemplate struct {
	At       time.Time
	Amount   alpacadecimal.Decimal
	Currency currencyx.Code
}

func (t TransferCustomerFBOToAccruedTemplate) Validate() error {
	if t.At.IsZero() {
		return fmt.Errorf("at is required")
	}

	if err := ledger.ValidateTransactionAmount(t.Amount); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	if err := ledger.ValidateCurrency(t.Currency); err != nil {
		return fmt.Errorf("currency: %w", err)
	}

	return nil
}

func (t TransferCustomerFBOToAccruedTemplate) typeGuard() guard {
	return true
}

var _ CustomerTransactionTemplate = (TransferCustomerFBOToAccruedTemplate{})

func (t TransferCustomerFBOToAccruedTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	collections, err := collectFromPrioritizedCustomerFBO(ctx, customerID, t.Currency, t.Amount, resolvers)
	if err != nil {
		return nil, fmt.Errorf("collect from prioritized FBO: %w", err)
	}
	if len(collections) == 0 {
		return nil, nil
	}

	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	accruedSubAccByCostBasis, err := resolveAccruedSubAccByCostBasis(ctx, customerAccounts.AccruedAccount, t.Currency, collections)
	if err != nil {
		return nil, err
	}

	return &TransactionInput{
		bookedAt:    t.At,
		entryInputs: buildRoutePreservingAccrualEntries(collections, accruedSubAccByCostBasis),
	}, nil
}

func resolveAccruedSubAccByCostBasis(
	ctx context.Context,
	accruedAccount ledger.CustomerAccruedAccount,
	currency currencyx.Code,
	collections []subAccountAmount,
) (map[string]subAccountAmount, error) {
	accruedSubAccByCostBasis := make(map[string]subAccountAmount, len(collections))

	for _, collection := range collections {
		key := costBasisKey(collection.subAccount.Route().CostBasis)
		current := accruedSubAccByCostBasis[key]
		if current.subAccount == nil {
			accruedSubAccount, err := accruedAccount.GetSubAccountForRoute(ctx, ledger.CustomerAccruedRouteParams{
				Currency:  currency,
				CostBasis: collection.subAccount.Route().CostBasis,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get accrued sub-account: %w", err)
			}
			current.subAccount = accruedSubAccount
		}

		current.amount = current.amount.Add(collection.amount)
		accruedSubAccByCostBasis[key] = current
	}

	return accruedSubAccByCostBasis, nil
}

func buildRoutePreservingAccrualEntries(
	collections []subAccountAmount,
	accruedSubAccByCostBasis map[string]subAccountAmount,
) []*EntryInput {
	entryInputs := make([]*EntryInput, 0, len(collections)*2)

	for _, collection := range collections {
		entryInputs = append(entryInputs, &EntryInput{
			address: collection.subAccount.Address(),
			amount:  collection.amount.Neg(),
		})
	}

	creditedCostBasis := make(map[string]struct{}, len(accruedSubAccByCostBasis))
	for _, collection := range collections {
		key := costBasisKey(collection.subAccount.Route().CostBasis)
		if _, ok := creditedCostBasis[key]; ok {
			continue
		}

		accrued := accruedSubAccByCostBasis[key]
		entryInputs = append(entryInputs, &EntryInput{
			address: accrued.subAccount.Address(),
			amount:  accrued.amount,
		})
		creditedCostBasis[key] = struct{}{}
	}

	return entryInputs
}

func costBasisKey(costBasis *alpacadecimal.Decimal) string {
	if costBasis == nil {
		return "null"
	}

	return costBasis.String()
}

// TransferCustomerFBOBucketToAccruedTemplate moves value from a specific customer FBO route
// into the matching accrued route. This is used for explicit advance-backed collection flows.
type TransferCustomerFBOBucketToAccruedTemplate struct {
	At             time.Time
	Amount         alpacadecimal.Decimal
	Currency       currencyx.Code
	CostBasis      *alpacadecimal.Decimal
	CreditPriority *int
}

func (t TransferCustomerFBOBucketToAccruedTemplate) Validate() error {
	if t.At.IsZero() {
		return fmt.Errorf("at is required")
	}

	if err := ledger.ValidateTransactionAmount(t.Amount); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	if err := ledger.ValidateCurrency(t.Currency); err != nil {
		return fmt.Errorf("currency: %w", err)
	}

	if t.CostBasis != nil {
		if err := ledger.ValidateCostBasis(*t.CostBasis); err != nil {
			return fmt.Errorf("cost basis: %w", err)
		}
	}

	if t.CreditPriority != nil {
		if err := ledger.ValidateCreditPriority(*t.CreditPriority); err != nil {
			return fmt.Errorf("credit priority: %w", err)
		}
	}

	return nil
}

func (t TransferCustomerFBOBucketToAccruedTemplate) typeGuard() guard {
	return true
}

var _ CustomerTransactionTemplate = (TransferCustomerFBOBucketToAccruedTemplate{})

func (t TransferCustomerFBOBucketToAccruedTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	priority := resolveCustomerFBOCreditPriority(t.CreditPriority)

	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	fbo, err := customerAccounts.FBOAccount.GetSubAccountForRoute(ctx, ledger.CustomerFBORouteParams{
		Currency:       t.Currency,
		CostBasis:      t.CostBasis,
		CreditPriority: priority,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get FBO sub-account: %w", err)
	}

	accrued, err := customerAccounts.AccruedAccount.GetSubAccountForRoute(ctx, ledger.CustomerAccruedRouteParams{
		Currency:  t.Currency,
		CostBasis: t.CostBasis,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get accrued sub-account: %w", err)
	}

	return &TransactionInput{
		bookedAt: t.At,
		entryInputs: []*EntryInput{
			{
				address: fbo.Address(),
				amount:  t.Amount.Neg(),
			},
			{
				address: accrued.Address(),
				amount:  t.Amount,
			},
		},
	}, nil
}

// TransferCustomerReceivableToAccruedTemplate acknowledges usage by moving it
// from receivable into the customer's accrued account.
type TransferCustomerReceivableToAccruedTemplate struct {
	At        time.Time
	Amount    alpacadecimal.Decimal
	Currency  currencyx.Code
	CostBasis *alpacadecimal.Decimal
}

func (t TransferCustomerReceivableToAccruedTemplate) Validate() error {
	if t.At.IsZero() {
		return fmt.Errorf("at is required")
	}

	if err := ledger.ValidateTransactionAmount(t.Amount); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	if err := ledger.ValidateCurrency(t.Currency); err != nil {
		return fmt.Errorf("currency: %w", err)
	}

	if t.CostBasis == nil {
		return fmt.Errorf("cost basis is required")
	}

	if err := ledger.ValidateCostBasis(*t.CostBasis); err != nil {
		return fmt.Errorf("cost basis: %w", err)
	}

	return nil
}

func (t TransferCustomerReceivableToAccruedTemplate) typeGuard() guard {
	return true
}

var _ CustomerTransactionTemplate = (TransferCustomerReceivableToAccruedTemplate{})

func (t TransferCustomerReceivableToAccruedTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	receivable, err := customerAccounts.ReceivableAccount.GetSubAccountForRoute(ctx, ledger.CustomerReceivableRouteParams{
		Currency:                       t.Currency,
		CostBasis:                      t.CostBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get receivable sub-account: %w", err)
	}

	accrued, err := customerAccounts.AccruedAccount.GetSubAccountForRoute(ctx, ledger.CustomerAccruedRouteParams{
		Currency:  t.Currency,
		CostBasis: t.CostBasis,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get accrued sub-account: %w", err)
	}

	return &TransactionInput{
		bookedAt: t.At,
		entryInputs: []*EntryInput{
			{
				address: receivable.Address(),
				amount:  t.Amount.Neg(),
			},
			{
				address: accrued.Address(),
				amount:  t.Amount,
			},
		},
	}, nil
}

// TranslateCustomerAccruedCostBasisTemplate moves accrued balance between cost-basis buckets
// without changing account type or currency.
type TranslateCustomerAccruedCostBasisTemplate struct {
	At            time.Time
	Amount        alpacadecimal.Decimal
	Currency      currencyx.Code
	FromCostBasis *alpacadecimal.Decimal
	ToCostBasis   *alpacadecimal.Decimal
}

func (t TranslateCustomerAccruedCostBasisTemplate) Validate() error {
	if t.At.IsZero() {
		return fmt.Errorf("at is required")
	}

	if err := ledger.ValidateTransactionAmount(t.Amount); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	if err := ledger.ValidateCurrency(t.Currency); err != nil {
		return fmt.Errorf("currency: %w", err)
	}

	if t.FromCostBasis != nil {
		if err := ledger.ValidateCostBasis(*t.FromCostBasis); err != nil {
			return fmt.Errorf("from cost basis: %w", err)
		}
	}

	if t.ToCostBasis == nil {
		return fmt.Errorf("to cost basis is required")
	}

	if err := ledger.ValidateCostBasis(*t.ToCostBasis); err != nil {
		return fmt.Errorf("to cost basis: %w", err)
	}

	if decimalPointersEqual(t.FromCostBasis, t.ToCostBasis) {
		return fmt.Errorf("from and to cost basis must differ")
	}

	return nil
}

func (t TranslateCustomerAccruedCostBasisTemplate) typeGuard() guard {
	return true
}

var _ CustomerTransactionTemplate = (TranslateCustomerAccruedCostBasisTemplate{})

func (t TranslateCustomerAccruedCostBasisTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	fromAccrued, err := customerAccounts.AccruedAccount.GetSubAccountForRoute(ctx, ledger.CustomerAccruedRouteParams{
		Currency:  t.Currency,
		CostBasis: t.FromCostBasis,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get source accrued sub-account: %w", err)
	}

	toAccrued, err := customerAccounts.AccruedAccount.GetSubAccountForRoute(ctx, ledger.CustomerAccruedRouteParams{
		Currency:  t.Currency,
		CostBasis: t.ToCostBasis,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get target accrued sub-account: %w", err)
	}

	return &TransactionInput{
		bookedAt: t.At,
		entryInputs: []*EntryInput{
			{
				address: fromAccrued.Address(),
				amount:  t.Amount.Neg(),
			},
			{
				address: toAccrued.Address(),
				amount:  t.Amount,
			},
		},
	}, nil
}
