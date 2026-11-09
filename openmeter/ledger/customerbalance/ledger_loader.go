package customerbalance

import (
	"context"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/creditvoid"
)

type ledgerCreditTransactionLoader struct {
	service  *service
	movement ledger.ListTransactionsCreditMovement
}

func newLedgerCreditTransactionLoader(s *service, movement ledger.ListTransactionsCreditMovement) creditTransactionLoader {
	return &ledgerCreditTransactionLoader{
		service:  s,
		movement: movement,
	}
}

func (l *ledgerCreditTransactionLoader) Load(ctx context.Context, input creditTransactionLoaderInput) (creditTransactionLoaderResult, error) {
	txs := make([]ledger.Transaction, 0, input.Limit+1)
	after := input.After
	before := input.Before
	hasMore := false

	for len(txs) <= input.Limit {
		result, err := l.service.Ledger.ListTransactions(ctx, ledger.ListTransactionsInput{
			Namespace: input.CustomerID.Namespace,
			Cursor:    after,
			Before:    before,
			Limit:     input.Limit,
			EntryFilter: ledger.TransactionEntryFilter{
				AccountIDs: []string{input.AccountID},
				Currency:   input.Currency,
				Route:      featureFilterRoute(input.FeatureFilter),
			},
			ReturnOnlyMatchingEntries: true,

			AsOf: &input.AsOf,

			CreditMovement: l.movement,
			ExcludeAnnotationFilters: map[string]string{
				ledger.AnnotationTransactionDirection:      string(ledger.TransactionDirectionCorrection),
				ledger.AnnotationCollectionType:            ledger.CollectionTypeBreakage,
				ledger.AnnotationCustomerBalanceVisibility: ledger.CustomerBalanceVisibilityInternal,
			},
		})
		if err != nil {
			return creditTransactionLoaderResult{}, err
		}

		page := result.Items
		if len(page) == 0 {
			break
		}

		if before != nil {
			// Scan nearest first so hidden rows can be replaced from the next page.
			page = slices.Clone(page)
			slices.Reverse(page)
		}

		for _, tx := range page {
			if _, ok := tx.Annotations()[creditvoid.AnnotationCreditVoidRecordID]; ok {
				continue
			}

			txs = append(txs, tx)
		}

		if len(txs) > input.Limit {
			hasMore = true
			break
		}

		if result.NextCursor == nil {
			break
		}

		if before != nil {
			cursor := page[len(page)-1].Cursor()
			before = &cursor
		} else {
			after = result.NextCursor
		}
	}

	if len(txs) > input.Limit {
		txs = txs[:input.Limit]
	}

	if input.Before != nil {
		slices.Reverse(txs)
	}

	items, err := creditTransactionsFromLedgerTransactions(txs)
	if err != nil {
		return creditTransactionLoaderResult{}, err
	}

	return creditTransactionLoaderResult{
		Items:   items,
		HasMore: hasMore,
	}, nil
}
