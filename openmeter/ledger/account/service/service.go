package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type service struct {
	repo   account.Repo
	locker *lockr.Locker
	live   account.AccountLiveServices
}

// New constructs an account Service. SubAccountService is self-wired so
// account-specific route helpers can create concrete posting addresses.
func New(repo account.Repo, locker *lockr.Locker) account.Service {
	svc := &service{
		repo:   repo,
		locker: locker,
	}
	svc.live = account.AccountLiveServices{SubAccountService: svc}
	return svc
}

var _ account.Service = (*service)(nil)

func (s *service) CreateAccount(ctx context.Context, input ledger.CreateAccountInput) (ledger.Account, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	accData, err := s.repo.CreateAccount(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return account.NewAccountFromData(*accData, s.live)
}

func (s *service) EnsureSubAccount(ctx context.Context, input ledger.CreateSubAccountInput) (ledger.SubAccount, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return transaction.Run(ctx, s.repo, func(ctx context.Context) (ledger.SubAccount, error) {
		subAccountData, err := s.repo.EnsureSubAccount(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to create sub-account: %w", err)
		}

		return account.NewSubAccountFromData(*subAccountData)
	})
}

func (s *service) GetAccountByID(ctx context.Context, id models.NamespacedID) (ledger.Account, error) {
	accData, err := s.repo.GetAccountByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return account.NewAccountFromData(*accData, s.live)
}

func (s *service) GetSubAccountByID(ctx context.Context, id models.NamespacedID) (ledger.SubAccount, error) {
	subAccountData, err := s.repo.GetSubAccountByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get sub-account: %w", err)
	}

	return account.NewSubAccountFromData(*subAccountData)
}

func (s *service) ListSubAccounts(ctx context.Context, input ledger.ListSubAccountsInput) ([]ledger.SubAccount, error) {
	subAccountDatas, err := s.repo.ListSubAccounts(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list sub-accounts: %w", err)
	}

	subAccounts := make([]ledger.SubAccount, 0, len(subAccountDatas))
	for _, subAccountData := range subAccountDatas {
		subAccount, err := account.NewSubAccountFromData(*subAccountData)
		if err != nil {
			return nil, fmt.Errorf("failed to map sub-account: %w", err)
		}
		subAccounts = append(subAccounts, subAccount)
	}

	return subAccounts, nil
}

func (s *service) ListAccounts(ctx context.Context, input ledger.ListAccountsInput) ([]ledger.Account, error) {
	accDatas, err := s.repo.ListAccounts(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	accounts := make([]ledger.Account, 0, len(accDatas))
	for _, accData := range accDatas {
		acc, err := account.NewAccountFromData(*accData, s.live)
		if err != nil {
			return nil, fmt.Errorf("failed to map account: %w", err)
		}
		accounts = append(accounts, acc)
	}

	return accounts, nil
}

func (s *service) LockAccountsForPosting(ctx context.Context, accounts []ledger.Account) error {
	if s.locker == nil {
		return nil
	}

	byID := make(map[models.NamespacedID]ledger.Account, len(accounts))
	for _, acc := range accounts {
		if acc == nil {
			continue
		}

		switch acc.Type() {
		case ledger.AccountTypeCustomerFBO, ledger.AccountTypeCustomerReceivable:
			byID[acc.ID()] = acc
		}
	}

	ids := make([]models.NamespacedID, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}

	sort.Slice(ids, func(i, j int) bool {
		if ids[i].Namespace == ids[j].Namespace {
			return ids[i].ID < ids[j].ID
		}

		return ids[i].Namespace < ids[j].Namespace
	})

	for _, id := range ids {
		key, err := lockr.NewKey("namespace", id.Namespace, "account", id.ID)
		if err != nil {
			return fmt.Errorf("failed to create lock key for account %s/%s: %w", id.Namespace, id.ID, err)
		}

		if err := s.locker.LockForTX(ctx, key); err != nil {
			return fmt.Errorf("failed to lock account %s/%s: %w", id.Namespace, id.ID, err)
		}
	}

	return nil
}
