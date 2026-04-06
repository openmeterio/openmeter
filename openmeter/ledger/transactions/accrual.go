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

func (t TransferCustomerFBOToAccruedTemplate) correct(_ context.Context, scope CorrectionInput, _ ResolverDependencies) ([]ledger.TransactionInput, error) {
	type selectedDebit struct {
		fboAddress     ledger.PostingAddress
		accruedAddress ledger.PostingAddress
		amount         alpacadecimal.Decimal
	}

	negativeFBOEntries := make([]ledger.Entry, 0)
	accruedAddressByCostBasis := make(map[string]ledger.PostingAddress)
	totalAvailable := alpacadecimal.Zero

	for _, entry := range scope.OriginalTransaction.Entries() {
		switch {
		case entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO && entry.Amount().IsNegative():
			negativeFBOEntries = append(negativeFBOEntries, entry)
			totalAvailable = totalAvailable.Add(entry.Amount().Abs())
		case entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerAccrued && entry.Amount().IsPositive():
			accruedAddressByCostBasis[costBasisKey(entry.PostingAddress().Route().Route().CostBasis)] = entry.PostingAddress()
		}
	}

	if scope.Amount.GreaterThan(totalAvailable) {
		return nil, fmt.Errorf("accrual correction amount %s exceeds original collected amount %s", scope.Amount.String(), totalAvailable.String())
	}

	selected := make([]selectedDebit, 0, len(negativeFBOEntries))
	remaining := scope.Amount
	for idx := len(negativeFBOEntries) - 1; idx >= 0 && remaining.IsPositive(); idx-- {
		entry := negativeFBOEntries[idx]
		amount := entry.Amount().Abs()
		if amount.GreaterThan(remaining) {
			amount = remaining
		}

		accruedAddress, ok := accruedAddressByCostBasis[costBasisKey(entry.PostingAddress().Route().Route().CostBasis)]
		if !ok {
			return nil, fmt.Errorf("missing accrued entry for FBO cost basis %s", costBasisKey(entry.PostingAddress().Route().Route().CostBasis))
		}

		selected = append(selected, selectedDebit{
			fboAddress:     entry.PostingAddress(),
			accruedAddress: accruedAddress,
			amount:         amount,
		})
		remaining = remaining.Sub(amount)
	}

	if remaining.IsPositive() {
		return nil, fmt.Errorf("accrual correction amount %s could not be fully allocated", scope.Amount.String())
	}

	accruedAmountsByAddress := make(map[string]selectedDebit)
	entryInputs := make([]*EntryInput, 0, len(selected)*2)
	for _, item := range selected {
		entryInputs = append(entryInputs, &EntryInput{
			address: item.fboAddress,
			amount:  item.amount,
		})

		key := item.accruedAddress.SubAccountID()
		current := accruedAmountsByAddress[key]
		current.accruedAddress = item.accruedAddress
		current.amount = current.amount.Add(item.amount)
		accruedAmountsByAddress[key] = current
	}

	for _, item := range accruedAmountsByAddress {
		entryInputs = append(entryInputs, &EntryInput{
			address: item.accruedAddress,
			amount:  item.amount.Neg(),
		})
	}

	return []ledger.TransactionInput{
		&TransactionInput{
			bookedAt:    scope.At,
			entryInputs: entryInputs,
		},
	}, nil
}

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

// TransferCustomerFBOAdvanceToAccruedTemplate moves value from the synthetic advance-backed
// customer FBO route into the matching accrued route.
type TransferCustomerFBOAdvanceToAccruedTemplate struct {
	At             time.Time
	Amount         alpacadecimal.Decimal
	Currency       currencyx.Code
	CostBasis      *alpacadecimal.Decimal
	CreditPriority *int
}

func (t TransferCustomerFBOAdvanceToAccruedTemplate) Validate() error {
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

func (t TransferCustomerFBOAdvanceToAccruedTemplate) typeGuard() guard {
	return true
}

var _ CustomerTransactionTemplate = (TransferCustomerFBOAdvanceToAccruedTemplate{})

func (t TransferCustomerFBOAdvanceToAccruedTemplate) correct(_ context.Context, scope CorrectionInput, _ ResolverDependencies) ([]ledger.TransactionInput, error) {
	var fboAddress ledger.PostingAddress
	var accruedAddress ledger.PostingAddress
	var fboAmount alpacadecimal.Decimal
	var accruedAmount alpacadecimal.Decimal

	for _, entry := range scope.OriginalTransaction.Entries() {
		switch {
		case entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO && entry.Amount().IsNegative():
			fboAddress = entry.PostingAddress()
			fboAmount = fboAmount.Add(entry.Amount().Abs())
		case entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerAccrued && entry.Amount().IsPositive():
			accruedAddress = entry.PostingAddress()
			accruedAmount = accruedAmount.Add(entry.Amount())
		}
	}

	if fboAddress == nil || accruedAddress == nil {
		return nil, fmt.Errorf("bucket accrual correction requires original FBO and accrued entries")
	}

	if scope.Amount.GreaterThan(fboAmount) || scope.Amount.GreaterThan(accruedAmount) {
		return nil, fmt.Errorf("bucket accrual correction amount %s exceeds original transaction amount", scope.Amount.String())
	}

	return []ledger.TransactionInput{
		&TransactionInput{
			bookedAt: scope.At,
			entryInputs: []*EntryInput{
				{
					address: fboAddress,
					amount:  scope.Amount,
				},
				{
					address: accruedAddress,
					amount:  scope.Amount.Neg(),
				},
			},
		},
	}, nil
}

func (t TransferCustomerFBOAdvanceToAccruedTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
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

func (t TransferCustomerReceivableToAccruedTemplate) correct(context.Context, CorrectionInput, ResolverDependencies) ([]ledger.TransactionInput, error) {
	return nil, templateCorrectionNotImplemented(templateName(t))
}

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

func (t TranslateCustomerAccruedCostBasisTemplate) correct(context.Context, CorrectionInput, ResolverDependencies) ([]ledger.TransactionInput, error) {
	return nil, templateCorrectionNotImplemented(templateName(t))
}

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
