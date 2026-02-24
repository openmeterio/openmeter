package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/models"
)

type service struct {
	repo account.Repo

	live account.AccountLiveServices
}

func New(repo account.Repo) account.Service {
	svc := &service{repo: repo}
	svc.live = account.AccountLiveServices{SubAccountService: svc}
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

func (s *service) CreateDimension(ctx context.Context, input account.CreateDimensionInput) (*account.DimensionData, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	res, err := s.repo.CreateDimension(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create dimension: %w", err)
	}

	return res, nil
}

func (s *service) CreateSubAccount(ctx context.Context, input account.CreateSubAccountInput) (*account.SubAccount, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	acc, err := s.GetAccountByID(ctx, models.NamespacedID{
		Namespace: input.Namespace,
		ID:        input.AccountID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	// Let's validate the provided dimensions
	if err := input.Dimensions.ValidateForAccountType(acc.Type()); err != nil {
		return nil, err
	}

	subAccountData, err := s.repo.CreateSubAccount(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create sub-account: %w", err)
	}

	return account.NewSubAccountFromData(*subAccountData)
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

	return account.NewSubAccountFromData(*subAccountData)
}

func (s *service) ListSubAccounts(ctx context.Context, input account.ListSubAccountsInput) ([]*account.SubAccount, error) {
	subAccountDatas, err := s.repo.ListSubAccounts(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list sub-accounts: %w", err)
	}

	subAccounts := make([]*account.SubAccount, 0, len(subAccountDatas))
	for _, subAccountData := range subAccountDatas {
		subAccount, err := account.NewSubAccountFromData(*subAccountData)
		if err != nil {
			return nil, fmt.Errorf("failed to map sub-account: %w", err)
		}
		subAccounts = append(subAccounts, subAccount)
	}

	return subAccounts, nil
}

func (s *service) GetDimensionByID(ctx context.Context, id models.NamespacedID) (*account.DimensionData, error) {
	return s.repo.GetDimensionByID(ctx, id)
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

func (s *service) GetDimensionByKeyAndValue(ctx context.Context, namespace string, key ledger.DimensionKey, value string) (*account.DimensionData, error) {
	return s.repo.GetDimensionByKeyAndValue(ctx, namespace, key, value)
}
