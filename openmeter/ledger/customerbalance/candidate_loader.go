package customerbalance

import (
	"context"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

// resolveCandidatePageFunc projects one page of ledger candidates into at most
// limit customer-visible rows, in candidate order.
type resolveCandidatePageFunc func(ctx context.Context, candidates []ledger.Transaction, limit int) ([]CreditTransaction, error)

type creditTransactionCandidatePage struct {
	items        []ledger.Transaction
	resumeCursor ledger.TransactionCursor
	hasMore      bool
}

// loadCandidateCreditTransactions traverses the customer balance ledger so rows
// projected from multiple transactions are ordered and paged by their final
// transaction cursors.
func (s *service) loadCandidateCreditTransactions(
	ctx context.Context,
	input creditTransactionLoaderInput,
	annotationFilters map[string]string,
	resolvePage resolveCandidatePageFunc,
) (creditTransactionLoaderResult, error) {
	items := make([]CreditTransaction, 0, input.Limit+1)
	after := input.After
	before := input.Before

	for len(items) <= input.Limit {
		page, err := s.listCandidatePage(ctx, input, annotationFilters, after, before)
		if err != nil {
			return creditTransactionLoaderResult{}, err
		}
		if len(page.items) == 0 {
			break
		}

		pageItems, err := resolvePage(ctx, page.items, input.Limit+1-len(items))
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

func (s *service) listCandidatePage(
	ctx context.Context,
	input creditTransactionLoaderInput,
	annotationFilters map[string]string,
	after, before *ledger.TransactionCursor,
) (creditTransactionCandidatePage, error) {
	accountIDs := []string{input.AccountID}
	if input.ReceivableAccountID != "" {
		accountIDs = append(accountIDs, input.ReceivableAccountID)
	}

	result, err := s.Ledger.ListTransactions(ctx, ledger.ListTransactionsInput{
		Namespace: input.CustomerID.Namespace,
		Cursor:    after,
		Before:    before,
		Limit:     max(chargeListPageSize, input.Limit+1),
		EntryFilter: ledger.TransactionEntryFilter{
			AccountIDs: accountIDs,
			Currency:   input.Currency,
			Route:      featureFilterRoute(input.FeatureFilter),
		},
		ReturnOnlyMatchingEntries: true,

		AsOf: &input.AsOf,

		AnnotationFilters: annotationFilters,
		ExcludeAnnotationFilters: map[string]string{
			ledger.AnnotationCollectionType:            ledger.CollectionTypeBreakage,
			ledger.AnnotationCustomerBalanceVisibility: ledger.CustomerBalanceVisibilityInternal,
		},
	})
	if err != nil {
		return creditTransactionCandidatePage{}, err
	}

	page := creditTransactionCandidatePage{
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
