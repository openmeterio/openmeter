package customerbalance

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/ledger"
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
	result, err := l.service.Ledger.ListTransactions(ctx, ledger.ListTransactionsInput{
		Namespace:      input.CustomerID.Namespace,
		Cursor:         input.After,
		Before:         input.Before,
		Limit:          input.Limit,
		AccountIDs:     []string{input.AccountID},
		Currency:       input.Currency,
		CreditMovement: l.movement,
		ExcludeAnnotationFilters: map[string]string{
			ledger.AnnotationCollectionType: ledger.CollectionTypeBreakage,
		},
	})
	if err != nil {
		return creditTransactionLoaderResult{}, err
	}

	items, err := creditTransactionsFromLedgerTransactions(result.Items)
	if err != nil {
		return creditTransactionLoaderResult{}, err
	}

	return creditTransactionLoaderResult{
		Items:   items,
		HasMore: result.NextCursor != nil,
	}, nil
}
