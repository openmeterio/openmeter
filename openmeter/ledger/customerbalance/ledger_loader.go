package customerbalance

import (
	"context"

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
			Namespace:      input.CustomerID.Namespace,
			Cursor:         after,
			Before:         before,
			Limit:          input.Limit + 1,
			AccountIDs:     []string{input.AccountID},
			Currency:       input.Currency,
			AsOf:           &input.AsOf,
			Route:          featureFilterRoute(input.FeatureFilter),
			CreditMovement: l.movement,
			ExcludeAnnotationFilters: map[string]string{
				ledger.AnnotationCollectionType: ledger.CollectionTypeBreakage,
			},
		})
		if err != nil {
			return creditTransactionLoaderResult{}, err
		}

		for _, tx := range result.Items {
			if _, ok := tx.Annotations()[creditvoid.AnnotationCreditVoidRecordID]; ok {
				continue
			}

			txs = append(txs, tx)
		}

		if result.NextCursor == nil {
			hasMore = false
			break
		}

		hasMore = true
		if len(txs) > input.Limit {
			break
		}

		if before != nil {
			before = result.NextCursor
		} else {
			after = result.NextCursor
		}
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
