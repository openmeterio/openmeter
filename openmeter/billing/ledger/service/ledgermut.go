package service

import (
	"context"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/billing/ledger"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/samber/lo/mutable"
)

type mutationService struct {
	adapter ledger.Adapter
	ledger  ledger.Ledger
}

var _ ledger.LedgerMutationService = (*mutationService)(nil)

func (s *Service) newMutationService(l ledger.Ledger) *mutationService {
	return &mutationService{
		adapter: s.adapter,
		ledger:  l,
	}
}

func (s *mutationService) Ledger() ledger.Ledger {
	return s.ledger
}

func (s *mutationService) UpsertSubledger(ctx context.Context, input ledger.UpsertSubledgerInput) (ledger.Subledger, error) {
	if err := input.Validate(); err != nil {
		return ledger.Subledger{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (ledger.Subledger, error) {
		return s.adapter.UpsertSubledger(ctx, ledger.UpsertSubledgerAdapterInput{
			UpsertSubledgerInput: input,
			LedgerID:             s.ledger.GetID(),
		})
	})
}

func (s *mutationService) CreateTransaction(ctx context.Context, input ledger.CreateTransactionInput) (ledger.Transaction, error) {
	if err := input.Validate(); err != nil {
		return ledger.Transaction{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (ledger.Transaction, error) {
		return s.adapter.CreateTransaction(ctx, input)
	})
}

func (s *mutationService) Withdraw(ctx context.Context, input ledger.WithdrawInput) (ledger.WithdrawalResults, error) {
	if err := input.Validate(); err != nil {
		return ledger.WithdrawalResults{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (ledger.WithdrawalResults, error) {
		adapterBal, err := s.adapter.GetBalance(ctx, s.ledger.GetID())
		if err != nil {
			return ledger.WithdrawalResults{}, err
		}

		bal := getBalanceResultFromAdapter(ctx, adapterBal)

		if bal.Balance.IsZero() {
			return ledger.WithdrawalResults{}, nil
		}

		toBeWithdrawn := input.Amount

		// Let's go from highest priority to lowest priority
		mutable.Reverse(bal.SubledgerBalances)

		transactions := make([]ledger.Transaction, 0, 1)

		for _, slBal := range bal.SubledgerBalances {
			if toBeWithdrawn.IsZero() {
				break
			}

			if slBal.Balance.IsZero() {
				continue
			}

			amountToWithdraw := alpacadecimal.Zero

			if toBeWithdrawn.GreaterThanOrEqual(slBal.Balance) {
				amountToWithdraw = slBal.Balance
			} else {
				amountToWithdraw = toBeWithdrawn
			}

			trns, err := s.adapter.CreateTransaction(ctx, ledger.CreateTransactionInput{
				Subledger:       slBal.Subledger,
				Amount:          amountToWithdraw.Neg(),
				TransactionMeta: input.TransactionMeta,
			})
			if err != nil {
				return ledger.WithdrawalResults{}, err
			}

			transactions = append(transactions, trns)

			toBeWithdrawn = toBeWithdrawn.Sub(amountToWithdraw)
		}

		return ledger.WithdrawalResults{
			Transactions:   transactions,
			TotalWithdrawn: input.Amount.Sub(toBeWithdrawn),
		}, nil
	})
}
