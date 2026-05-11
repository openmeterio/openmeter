package transactions

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// TransferCustomerFBOToAccruedTemplate moves preselected customer FBO value
// into the customer's accrued account.
type TransferCustomerFBOToAccruedTemplate struct {
	At       time.Time
	Currency currencyx.Code
	Sources  []PostingAmount
}

func (t TransferCustomerFBOToAccruedTemplate) Validate() error {
	if t.At.IsZero() {
		return fmt.Errorf("at is required")
	}

	if err := ledger.ValidateCurrency(t.Currency); err != nil {
		return fmt.Errorf("currency: %w", err)
	}

	for i, source := range t.Sources {
		if source.Address == nil {
			return fmt.Errorf("sources[%d]: address is required", i)
		}

		if source.Address.AccountType() != ledger.AccountTypeCustomerFBO {
			return fmt.Errorf("sources[%d]: account type must be customer_fbo", i)
		}

		if source.Address.Route().Route().Currency != t.Currency {
			return fmt.Errorf("sources[%d]: currency must be %s", i, t.Currency)
		}

		if err := ledger.ValidateTransactionAmount(source.Amount); err != nil {
			return fmt.Errorf("sources[%d].amount: %w", i, err)
		}
	}

	return nil
}

func (t TransferCustomerFBOToAccruedTemplate) typeGuard() guard {
	return true
}

func (t TransferCustomerFBOToAccruedTemplate) code() TransactionTemplateCode {
	return TemplateCodeTransferCustomerFBOToAccrued
}

var _ CustomerTransactionTemplate = (TransferCustomerFBOToAccruedTemplate{})

func (t TransferCustomerFBOToAccruedTemplate) correct(scope CorrectionInput) ([]ledger.TransactionInput, error) {
	negativeFBOEntries := make([]ledger.Entry, 0)
	positiveAccruedEntries := make([]ledger.Entry, 0)

	for _, entry := range scope.OriginalTransaction.Entries() {
		switch {
		case entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO && entry.Amount().IsNegative():
			negativeFBOEntries = append(negativeFBOEntries, entry)
		case entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerAccrued && entry.Amount().IsPositive():
			positiveAccruedEntries = append(positiveAccruedEntries, entry)
		}
	}

	slices.SortStableFunc(negativeFBOEntries, compareFBOAccrualCorrectionSourceEntries)
	postings, err := allocateCorrectionLegs(
		negativeFBOEntries,
		positiveAccruedEntries,
		t.routePairingKey,
		func(entry ledger.Entry) alpacadecimal.Decimal {
			return entry.Amount().Abs()
		},
		scope.Amount,
	)
	if err != nil {
		return nil, fmt.Errorf("allocate accrual correction legs: %w", err)
	}

	return []ledger.TransactionInput{
		&TransactionInput{
			bookedAt:    scope.At,
			entryInputs: mapCorrectionPostingsToEntryInputs(postings),
		},
	}, nil
}

func (t TransferCustomerFBOToAccruedTemplate) routePairingKey(address ledger.PostingAddress) routePairingKey {
	route := address.Route().Route()

	return routePairingKey{
		currency:  route.Currency,
		costBasis: costBasisKey(route.CostBasis),
	}
}

func compareFBOAccrualCorrectionSourceEntries(left ledger.Entry, right ledger.Entry) int {
	leftPriority := fboCorrectionPriority(left.PostingAddress())
	rightPriority := fboCorrectionPriority(right.PostingAddress())
	if leftPriority != rightPriority {
		return cmp.Compare(leftPriority, rightPriority)
	}

	return cmp.Compare(left.PostingAddress().SubAccountID(), right.PostingAddress().SubAccountID())
}

func fboCorrectionPriority(address ledger.PostingAddress) int {
	priority := address.Route().Route().CreditPriority
	if priority == nil {
		return ledger.DefaultCustomerFBOPriority
	}

	return *priority
}

func (t TransferCustomerFBOToAccruedTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	if len(t.Sources) == 0 {
		return nil, nil
	}

	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	accruedSubAccByKey, err := t.resolveAccruedSubAccByRoutePairingKey(ctx, customerAccounts.AccruedAccount, t.Sources)
	if err != nil {
		return nil, err
	}

	return &TransactionInput{
		bookedAt:    t.At,
		entryInputs: t.buildRoutePreservingAccrualEntries(t.Sources, accruedSubAccByKey),
	}, nil
}

func (t TransferCustomerFBOToAccruedTemplate) resolveAccruedSubAccByRoutePairingKey(
	ctx context.Context,
	accruedAccount ledger.CustomerAccruedAccount,
	sources []PostingAmount,
) (map[routePairingKey]PostingAmount, error) {
	accruedSubAccByKey := make(map[routePairingKey]PostingAmount, len(sources))

	for _, source := range sources {
		key := t.routePairingKey(source.Address)
		current := accruedSubAccByKey[key]
		if current.Address == nil {
			accruedSubAccount, err := accruedAccount.GetSubAccountForRoute(ctx, ledger.CustomerAccruedRouteParams{
				Currency:  t.Currency,
				CostBasis: source.Address.Route().Route().CostBasis,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get accrued sub-account: %w", err)
			}
			current.Address = accruedSubAccount.Address()
		}

		current.Amount = current.Amount.Add(source.Amount)
		accruedSubAccByKey[key] = current
	}

	return accruedSubAccByKey, nil
}

func (t TransferCustomerFBOToAccruedTemplate) buildRoutePreservingAccrualEntries(
	sources []PostingAmount,
	accruedSubAccByKey map[routePairingKey]PostingAmount,
) []*EntryInput {
	entryInputs := make([]*EntryInput, 0, len(sources)*2)

	for _, source := range sources {
		entryInputs = append(entryInputs, &EntryInput{
			address: source.Address,
			amount:  source.Amount.Neg(),
		})
	}

	creditedKeys := make(map[routePairingKey]struct{}, len(accruedSubAccByKey))
	for _, source := range sources {
		key := t.routePairingKey(source.Address)
		if _, ok := creditedKeys[key]; ok {
			continue
		}

		accrued := accruedSubAccByKey[key]
		entryInputs = append(entryInputs, &EntryInput{
			address: accrued.Address,
			amount:  accrued.Amount,
		})
		creditedKeys[key] = struct{}{}
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

func (t TransferCustomerFBOAdvanceToAccruedTemplate) code() TransactionTemplateCode {
	return TemplateCodeTransferCustomerFBOAdvanceToAccrued
}

var _ CustomerTransactionTemplate = (TransferCustomerFBOAdvanceToAccruedTemplate{})

func (t TransferCustomerFBOAdvanceToAccruedTemplate) correct(scope CorrectionInput) ([]ledger.TransactionInput, error) {
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

func (t TransferCustomerReceivableToAccruedTemplate) code() TransactionTemplateCode {
	return TemplateCodeTransferCustomerReceivableToAccrued
}

var _ CustomerTransactionTemplate = (TransferCustomerReceivableToAccruedTemplate{})

func (t TransferCustomerReceivableToAccruedTemplate) correct(CorrectionInput) ([]ledger.TransactionInput, error) {
	return nil, templateCorrectionNotImplemented(TemplateCode(t))
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

func (t TranslateCustomerAccruedCostBasisTemplate) code() TransactionTemplateCode {
	return TemplateCodeTranslateCustomerAccruedCostBasis
}

var _ CustomerTransactionTemplate = (TranslateCustomerAccruedCostBasisTemplate{})

func (t TranslateCustomerAccruedCostBasisTemplate) correct(scope CorrectionInput) ([]ledger.TransactionInput, error) {
	var fromAccruedAddress ledger.PostingAddress
	var toAccruedAddress ledger.PostingAddress
	var fromAccruedAmount alpacadecimal.Decimal
	var toAccruedAmount alpacadecimal.Decimal

	for _, entry := range scope.OriginalTransaction.Entries() {
		switch {
		case entry.PostingAddress().AccountType() != ledger.AccountTypeCustomerAccrued:
			continue
		case entry.Amount().IsNegative():
			fromAccruedAddress = entry.PostingAddress()
			fromAccruedAmount = fromAccruedAmount.Add(entry.Amount().Abs())
		case entry.Amount().IsPositive():
			toAccruedAddress = entry.PostingAddress()
			toAccruedAmount = toAccruedAmount.Add(entry.Amount())
		}
	}

	if fromAccruedAddress == nil || toAccruedAddress == nil {
		return nil, fmt.Errorf("accrued cost-basis translation correction requires original accrued entries")
	}

	if scope.Amount.GreaterThan(fromAccruedAmount) || scope.Amount.GreaterThan(toAccruedAmount) {
		return nil, fmt.Errorf("accrued cost-basis translation correction amount %s exceeds original transaction amount", scope.Amount.String())
	}

	return []ledger.TransactionInput{
		&TransactionInput{
			bookedAt: scope.At,
			entryInputs: []*EntryInput{
				{
					address: fromAccruedAddress,
					amount:  scope.Amount,
				},
				{
					address: toAccruedAddress,
					amount:  scope.Amount.Neg(),
				},
			},
		},
	}, nil
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
