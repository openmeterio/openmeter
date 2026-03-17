package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type service struct {
	repo account.Repo

	live account.AccountLiveServices
}

// New constructs an account Service. live carries the external runtime dependencies
// (Locker, Querier); SubAccountService is always self-wired and will be overwritten.
func New(repo account.Repo, live account.AccountLiveServices) account.Service {
	svc := &service{repo: repo}
	live.SubAccountService = svc
	svc.live = live
	return svc
}

var _ account.Service = (*service)(nil)

func (s *service) CreateAccount(ctx context.Context, input account.CreateAccountInput) (*account.Account, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	accData, err := s.repo.CreateAccount(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return account.NewAccountFromData(*accData, s.live)
}

func (s *service) EnsureSubAccount(ctx context.Context, input account.CreateSubAccountInput) (*account.SubAccount, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return transaction.Run(ctx, s.repo, func(ctx context.Context) (*account.SubAccount, error) {
		subAccountData, err := s.repo.EnsureSubAccount(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to create sub-account: %w", err)
		}

		acc, err := s.GetAccountByID(ctx, models.NamespacedID{
			Namespace: input.Namespace,
			ID:        input.AccountID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get parent account: %w", err)
		}

		return account.NewSubAccountFromData(*subAccountData, acc)
	})
}

func (s *service) GetAccountByID(ctx context.Context, id models.NamespacedID) (*account.Account, error) {
	accData, err := s.repo.GetAccountByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return account.NewAccountFromData(*accData, s.live)
}

func (s *service) GetSubAccountByID(ctx context.Context, id models.NamespacedID) (*account.SubAccount, error) {
	subAccountData, err := s.repo.GetSubAccountByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get sub-account: %w", err)
	}

	acc, err := s.GetAccountByID(ctx, models.NamespacedID{
		Namespace: id.Namespace,
		ID:        subAccountData.AccountID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get parent account: %w", err)
	}

	return account.NewSubAccountFromData(*subAccountData, acc)
}

func (s *service) ListSubAccounts(ctx context.Context, input account.ListSubAccountsInput) ([]*account.SubAccount, error) {
	subAccountDatas, err := s.repo.ListSubAccounts(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list sub-accounts: %w", err)
	}

	// FIXME: this will be a problem
	subAccounts := make([]*account.SubAccount, 0, len(subAccountDatas))
	accountsByID := make(map[string]*account.Account, len(subAccountDatas))
	for _, subAccountData := range subAccountDatas {
		acc, ok := accountsByID[subAccountData.AccountID]
		if !ok {
			acc, err = s.GetAccountByID(ctx, models.NamespacedID{
				Namespace: subAccountData.Namespace,
				ID:        subAccountData.AccountID,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get parent account: %w", err)
			}

			accountsByID[subAccountData.AccountID] = acc
		}

		subAccount, err := account.NewSubAccountFromData(*subAccountData, acc)
		if err != nil {
			return nil, fmt.Errorf("failed to map sub-account: %w", err)
		}
		subAccounts = append(subAccounts, subAccount)
	}

	return subAccounts, nil
}

func (s *service) ListAccounts(ctx context.Context, input account.ListAccountsInput) ([]*account.Account, error) {
	accDatas, err := s.repo.ListAccounts(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	accounts := make([]*account.Account, 0, len(accDatas))
	for _, accData := range accDatas {
		acc, err := account.NewAccountFromData(*accData, s.live)
		if err != nil {
			return nil, fmt.Errorf("failed to map account: %w", err)
		}
		accounts = append(accounts, acc)
	}

	return accounts, nil
}
