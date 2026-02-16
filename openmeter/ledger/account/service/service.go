package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/models"
)

type service struct {
	repo    account.Repo
	querier ledger.Querier
}

func New(repo account.Repo) account.Service {
	return &service{
		repo: repo,
	}
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

	return account.NewAccountFromData(s.querier, *accData)
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

	return account.NewAccountFromData(s.querier, *accData)
}

func (s *service) GetSubAccountByID(ctx context.Context, id models.NamespacedID) (*account.SubAccount, error) {
	subAccountData, err := s.repo.GetSubAccountByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get sub-account: %w", err)
	}

	return account.NewSubAccountFromData(*subAccountData)
}

func (s *service) GetDimensionByID(ctx context.Context, id models.NamespacedID) (*account.DimensionData, error) {
	return s.repo.GetDimensionByID(ctx, id)
}
