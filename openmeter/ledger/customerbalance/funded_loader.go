package customerbalance

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

type fundedCreditTransactionLoader struct {
	service *service
}

type fundedCandidatePage struct {
	items        []ledger.Transaction
	resumeCursor ledger.TransactionCursor
	hasMore      bool
}

func newFundedCreditTransactionLoader(s *service) creditTransactionLoader {
	return &fundedCreditTransactionLoader{service: s}
}

func (l *fundedCreditTransactionLoader) Load(ctx context.Context, input creditTransactionLoaderInput) (creditTransactionLoaderResult, error) {
	items := make([]CreditTransaction, 0, input.Limit+1)
	after := input.After
	before := input.Before

	// Traverse the customer balance ledger so effective-time rows are ordered
	// and paged by their final transaction cursors. Charges are only used to
	// identify and hydrate the credit purchases behind those candidates.
	for len(items) <= input.Limit {
		page, err := l.listCandidatePage(ctx, input, after, before)
		if err != nil {
			return creditTransactionLoaderResult{}, err
		}
		if len(page.items) == 0 {
			break
		}

		chargesByID, err := l.hydrateCandidateCharges(ctx, input.CustomerID.Namespace, page.items)
		if err != nil {
			return creditTransactionLoaderResult{}, err
		}

		pageItems, err := l.resolveCandidatePage(
			ctx,
			input,
			page.items,
			chargesByID,
			input.Limit+1-len(items),
		)
		if err != nil {
			return creditTransactionLoaderResult{}, err
		}
		items = append(items, pageItems...)

		if len(items) > input.Limit || !page.hasMore {
			break
		}

		if before != nil {
			before = &page.resumeCursor
		} else {
			after = &page.resumeCursor
		}
	}

	hasMore := len(items) > input.Limit
	if hasMore {
		items = items[:input.Limit]
	}
	if input.Before != nil {
		slices.Reverse(items)
	}

	return creditTransactionLoaderResult{
		Items:   items,
		HasMore: hasMore,
	}, nil
}

func (l *fundedCreditTransactionLoader) listCandidatePage(
	ctx context.Context,
	input creditTransactionLoaderInput,
	after, before *ledger.TransactionCursor,
) (fundedCandidatePage, error) {
	accountIDs := []string{input.AccountID}
	if input.ReceivableAccountID != "" {
		accountIDs = append(accountIDs, input.ReceivableAccountID)
	}

	result, err := l.service.Ledger.ListTransactions(ctx, ledger.ListTransactionsInput{
		Namespace:  input.CustomerID.Namespace,
		Cursor:     after,
		Before:     before,
		Limit:      max(chargeListPageSize, input.Limit+1),
		AccountIDs: accountIDs,
		Currency:   input.Currency,
		AsOf:       &input.AsOf,
		Route:      featureFilterRoute(input.FeatureFilter),
		ExcludeAnnotationFilters: map[string]string{
			ledger.AnnotationCollectionType: ledger.CollectionTypeBreakage,
		},
	})
	if err != nil {
		return fundedCandidatePage{}, err
	}

	page := fundedCandidatePage{
		items:   result.Items,
		hasMore: result.NextCursor != nil,
	}
	if len(page.items) == 0 {
		return page, nil
	}

	if before != nil {
		// The ledger returns before-pages newest-first. Scan the nearest newer
		// candidate first, then resume from the page's newest edge.
		page.resumeCursor = page.items[0].Cursor()
		page.items = slices.Clone(page.items)
		slices.Reverse(page.items)
	} else {
		page.resumeCursor = page.items[len(page.items)-1].Cursor()
	}

	return page, nil
}

func (l *fundedCreditTransactionLoader) hydrateCandidateCharges(
	ctx context.Context,
	namespace string,
	candidates []ledger.Transaction,
) (map[string]charges.Charge, error) {
	chargeIDs := lo.Uniq(lo.FilterMap(candidates, func(candidate ledger.Transaction, _ int) (string, bool) {
		chargeID := chargeIDFromAnnotations(candidate.Annotations())

		return chargeID, chargeID != ""
	}))
	if len(chargeIDs) == 0 {
		return map[string]charges.Charge{}, nil
	}

	hydratedCharges, err := l.service.ChargesService.GetByIDs(ctx, charges.GetByIDsInput{
		Namespace: namespace,
		IDs:       chargeIDs,
		Expands:   meta.Expands{meta.ExpandRealizations},
	})
	if err != nil {
		return nil, fmt.Errorf("hydrate funded credit transactions: %w", err)
	}

	chargesByID := make(map[string]charges.Charge, len(hydratedCharges))
	for _, hydratedCharge := range hydratedCharges {
		chargeID, err := hydratedCharge.GetChargeID()
		if err != nil {
			return nil, fmt.Errorf("read hydrated charge ID: %w", err)
		}

		chargesByID[chargeID.ID] = hydratedCharge
	}

	return chargesByID, nil
}

func (l *fundedCreditTransactionLoader) resolveCandidatePage(
	ctx context.Context,
	input creditTransactionLoaderInput,
	candidates []ledger.Transaction,
	chargesByID map[string]charges.Charge,
	limit int,
) ([]CreditTransaction, error) {
	items := make([]CreditTransaction, 0, min(len(candidates), limit))
	resolvedByGroupID := make(map[string][]CreditTransaction)
	for _, candidate := range candidates {
		item, ok, err := l.resolveCandidate(ctx, input, candidate, chargesByID, resolvedByGroupID)
		if err != nil {
			return nil, err
		}
		if ok {
			items = append(items, item)
		}
		if len(items) == limit {
			break
		}
	}

	return items, nil
}

func (l *fundedCreditTransactionLoader) resolveCandidate(
	ctx context.Context,
	input creditTransactionLoaderInput,
	candidate ledger.Transaction,
	chargesByID map[string]charges.Charge,
	resolvedByGroupID map[string][]CreditTransaction,
) (CreditTransaction, bool, error) {
	chargeID := chargeIDFromAnnotations(candidate.Annotations())
	hydratedCharge, ok := chargesByID[chargeID]
	if !ok || hydratedCharge.Type() != meta.ChargeTypeCreditPurchase {
		return CreditTransaction{}, false, nil
	}

	charge, err := hydratedCharge.AsCreditPurchaseCharge()
	if err != nil {
		return CreditTransaction{}, false, fmt.Errorf("map funded credit transaction charge: %w", err)
	}
	if charge.Intent.CustomerID != input.CustomerID.ID {
		return CreditTransaction{}, false, nil
	}

	item, ok := fundedCreditTransactionFromCharge(charge)
	if !ok {
		return CreditTransaction{}, false, nil
	}
	if input.Currency != nil && item.Currency != *input.Currency {
		return CreditTransaction{}, false, nil
	}

	resolvedItems, ok := resolvedByGroupID[item.fundedTransactionGroupID]
	if !ok {
		resolvedItems, err = l.resolveBalances(ctx, input, item)
		if err != nil {
			return CreditTransaction{}, false, err
		}
		resolvedByGroupID[item.fundedTransactionGroupID] = resolvedItems
	}

	candidateCursor := candidate.Cursor()
	for _, resolvedItem := range resolvedItems {
		// Same-time impacts are emitted once, at the latest contributing
		// transaction cursor selected by resolveBalances.
		if resolvedItem.balanceCursor != nil &&
			resolvedItem.balanceCursor.Compare(candidateCursor) == 0 &&
			creditTransactionMatchesCursorWindow(resolvedItem, input.After, input.Before) {
			return resolvedItem, true, nil
		}
	}

	return CreditTransaction{}, false, nil
}

func fundedCreditTransactionFromCharge(charge creditpurchase.Charge) (CreditTransaction, bool) {
	grant := charge.Realizations.CreditGrantRealization
	if grant == nil || grant.TransactionGroupID == "" {
		return CreditTransaction{}, false
	}

	return CreditTransaction{
		ID:          models.NamespacedID(charge.GetChargeID()),
		CreatedAt:   charge.CreatedAt,
		BookedAt:    grant.Time,
		Type:        CreditTransactionTypeFunded,
		GrantVoided: charge.State.VoidedAt != nil,
		Currency:    charge.Intent.Currency.GetCode(),
		Amount:      charge.Intent.CreditAmount,
		Name:        charge.Intent.Name,
		Description: charge.Intent.Description,
		Annotations: models.Annotations{
			ledger.AnnotationChargeID: charge.ID,
		},
		currencyReference:        charge.Intent.Currency.Reference(),
		fundedTransactionGroupID: grant.TransactionGroupID,
	}, true
}

func creditTransactionMatchesCursorWindow(item CreditTransaction, after, before *ledger.TransactionCursor) bool {
	cursor := creditTransactionCursor(item)
	if after != nil && cursor.Compare(*after) >= 0 {
		return false
	}
	if before != nil && cursor.Compare(*before) <= 0 {
		return false
	}

	return true
}

// resolveBalances splits a funded transaction into separate balance movements
// when its ledger impacts have different booked times.
func (l *fundedCreditTransactionLoader) resolveBalances(
	ctx context.Context,
	input creditTransactionLoaderInput,
	item CreditTransaction,
) ([]CreditTransaction, error) {
	if item.fundedTransactionGroupID == "" {
		return []CreditTransaction{item}, nil
	}

	// Get the complete ledger group backing the funded activity.
	group, err := l.service.Ledger.GetTransactionGroup(ctx, models.NamespacedID{
		Namespace: input.CustomerID.Namespace,
		ID:        item.fundedTransactionGroupID,
	})
	if err != nil {
		return nil, fmt.Errorf("get funded credit transaction group %s: %w", item.fundedTransactionGroupID, err)
	}

	// Validate the unfiltered group against the nominal funded amount.
	unfilteredImpacts, err := fundedCreditTransactionBalanceImpacts(group, GetBalanceServiceInput{
		Currency:          item.Currency,
		currencyReference: item.balanceCurrencyReference(),
	})
	if err != nil {
		return nil, fmt.Errorf("resolve funded credit transaction group %s balance impacts: %w", item.fundedTransactionGroupID, err)
	}
	if len(unfilteredImpacts) == 0 {
		return nil, fmt.Errorf("funded credit transaction group %s has no customer balance impact", item.fundedTransactionGroupID)
	}

	total := alpacadecimal.Zero
	for _, impact := range unfilteredImpacts {
		total = total.Add(impact.Amount)
	}
	if !total.Equal(item.Amount) {
		return nil, fmt.Errorf(
			"funded credit transaction group %s customer balance impact %s does not match funded amount %s",
			item.fundedTransactionGroupID,
			total,
			item.Amount,
		)
	}

	// Calculate the impacts visible in the requested balance projection.
	impacts := unfilteredImpacts
	if input.FeatureFilter.IsPresent() {
		impacts, err = fundedCreditTransactionBalanceImpacts(group, GetBalanceServiceInput{
			Currency:          item.Currency,
			FeatureFilter:     input.FeatureFilter,
			currencyReference: item.balanceCurrencyReference(),
		})
		if err != nil {
			return nil, fmt.Errorf("resolve funded credit transaction group %s filtered balance impacts: %w", item.fundedTransactionGroupID, err)
		}
	}

	items := make([]CreditTransaction, 0, len(impacts))
	for _, impact := range impacts {
		if impact.Cursor.BookedAt.After(input.AsOf) {
			continue
		}

		resolvedItem := item
		resolvedItem.BookedAt = impact.Cursor.BookedAt
		resolvedItem.Amount = impact.Amount
		resolvedItem.balanceCursor = &impact.Cursor
		resolvedItem.balanceImpact = &impact.Amount
		resolvedItem.currencyReference = impact.CurrencyReference
		items = append(items, resolvedItem)
	}

	return items, nil
}

type fundedCreditTransactionBalanceImpact struct {
	Amount            alpacadecimal.Decimal
	Cursor            ledger.TransactionCursor
	CurrencyReference currencies.CurrencyReference
}

// fundedCreditTransactionBalanceImpacts follows the same two ledger components
// as settled balance and groups them by the time they affect that balance.
// Looking at the actual scoped entries keeps legacy credit purchase groups
// readable without relying on transaction template annotations.
func fundedCreditTransactionBalanceImpacts(group ledger.TransactionGroup, input GetBalanceServiceInput) ([]fundedCreditTransactionBalanceImpact, error) {
	impactsByBookedAt := make(map[time.Time]fundedCreditTransactionBalanceImpact)
	var groupCurrency currencies.CurrencyReference

	for _, tx := range group.Transactions() {
		if tx.Annotations()[ledger.AnnotationCollectionType] == ledger.CollectionTypeBreakage {
			continue
		}

		impact, currencyReference, err := fundedCreditTransactionImpact(tx, input)
		if err != nil {
			return nil, err
		}
		if impact.IsZero() {
			continue
		}
		if groupCurrency.Code == "" {
			groupCurrency = currencyReference
		} else if !groupCurrency.Equal(currencyReference) {
			return nil, fmt.Errorf("transactions have multiple customer balance currencies")
		}

		cursor := tx.Cursor()
		bookedAt := cursor.BookedAt.UTC()
		groupedImpact, ok := impactsByBookedAt[bookedAt]
		if !ok {
			groupedImpact.Amount = alpacadecimal.Zero
			groupedImpact.CurrencyReference = currencyReference
		}
		groupedImpact.Amount = groupedImpact.Amount.Add(impact)
		if groupedImpact.Cursor.BookedAt.IsZero() || groupedImpact.Cursor.Compare(cursor) < 0 {
			groupedImpact.Cursor = cursor
		}
		impactsByBookedAt[bookedAt] = groupedImpact
	}

	impacts := make([]fundedCreditTransactionBalanceImpact, 0, len(impactsByBookedAt))
	for _, impact := range impactsByBookedAt {
		if !impact.Amount.IsZero() {
			impacts = append(impacts, impact)
		}
	}
	slices.SortFunc(impacts, func(a, b fundedCreditTransactionBalanceImpact) int {
		return b.Cursor.Compare(a.Cursor)
	})

	return impacts, nil
}

func fundedCreditTransactionImpact(tx ledger.Transaction, input GetBalanceServiceInput) (alpacadecimal.Decimal, currencies.CurrencyReference, error) {
	bookedFilter := ledger.ImpactFilter{
		AccountType: ledger.AccountTypeCustomerFBO,
		Route:       input.bookedRoute(),
	}
	advanceFilter := ledger.ImpactFilter{
		AccountType: ledger.AccountTypeCustomerReceivable,
		Route:       input.advanceRoute(),
	}

	impact := alpacadecimal.Zero
	var currencyReference currencies.CurrencyReference
	for _, entry := range tx.Entries() {
		if !ledger.EntryMatchesImpactFilter(entry, bookedFilter) && !ledger.EntryMatchesImpactFilter(entry, advanceFilter) {
			continue
		}

		entryCurrency := entry.PostingAddress().Route().Route().Currency
		if currencyReference.Code == "" {
			currencyReference = entryCurrency
		} else if !currencyReference.Equal(entryCurrency) {
			return alpacadecimal.Zero, currencies.CurrencyReference{}, fmt.Errorf("transaction %s has multiple customer balance currencies", tx.ID().ID)
		}
		impact = impact.Add(entry.Amount())
	}

	return impact, currencyReference, nil
}
