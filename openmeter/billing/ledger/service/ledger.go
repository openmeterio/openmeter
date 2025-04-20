package service

import (
	"context"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/billing/ledger"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

var _ ledger.LedgerService = (*Service)(nil)

func (s *Service) WithLockedLedger(ctx context.Context, input ledger.WithLockedLedgerInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		return s.adapter.WithLockedLedger(ctx, ledger.WithLockedLedgerAdapterInput{
			Customer: input.Customer,
			Currency: input.Currency,
			Callback: func(ctx context.Context, ledger ledger.Ledger) error {
				return input.Callback(ctx, s.newMutationService(ledger))
			},
		})
	})
}

func (s *Service) GetBalance(ctx context.Context, input ledger.GetBalanceInput) (ledger.GetBalanceResult, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (ledger.GetBalanceResult, error) {
		l, err := s.adapter.GetLedger(ctx, input)
		if err != nil {
			return ledger.GetBalanceResult{}, err
		}

		bal, err := s.adapter.GetBalance(ctx, l.GetID())
		if err != nil {
			return ledger.GetBalanceResult{}, err
		}

		return getBalanceResultFromAdapter(ctx, bal), nil
	})
}

func getBalanceResultFromAdapter(ctx context.Context, bal ledger.GetBalanceAdapterResult) ledger.GetBalanceResult {
	totalBalance := alpacadecimal.Zero
	for _, subledger := range bal {
		totalBalance = totalBalance.Add(subledger.Balance)
	}

	slices.SortFunc(bal, func(a, b ledger.SubledgerBalance) int {
		cmp := int(a.Subledger.Priority - b.Subledger.Priority)
		if cmp != 0 {
			return cmp
		}

		return int(a.Subledger.CreatedAt.Sub(b.Subledger.CreatedAt).Seconds())
	})

	return ledger.GetBalanceResult{
		Balance:           totalBalance,
		SubledgerBalances: bal,
	}
}
