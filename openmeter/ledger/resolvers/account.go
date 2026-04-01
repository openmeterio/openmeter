package resolvers

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

// AccountResolver implements ledger.AccountResolver and manages customer ledger account provisioning.
type AccountResolver struct {
	AccountService ledgeraccount.Service
	Repo           CustomerAccountRepo
	Locker         *lockr.Locker
}

// AccountResolverConfig holds the dependencies for constructing a Service.
type AccountResolverConfig struct {
	AccountService ledgeraccount.Service
	Repo           CustomerAccountRepo
	Locker         *lockr.Locker
}

// NewAccountResolver constructs a resolvers Service from the given config.
func NewAccountResolver(cfg AccountResolverConfig) *AccountResolver {
	return &AccountResolver{
		AccountService: cfg.AccountService,
		Repo:           cfg.Repo,
		Locker:         cfg.Locker,
	}
}

var _ ledger.AccountResolver = (*AccountResolver)(nil)

// CreateCustomerAccounts creates FBO and Receivable ledger accounts for a new customer
// and stores the mappings in the linking table.
func (s *AccountResolver) CreateCustomerAccounts(ctx context.Context, customerID customer.CustomerID) (ledger.CustomerAccounts, error) {
	return transaction.Run(ctx, s.Repo, func(ctx context.Context) (ledger.CustomerAccounts, error) {
		if err := s.lockCustomerProvisioning(ctx, customerID); err != nil {
			return ledger.CustomerAccounts{}, fmt.Errorf("failed to lock customer account provisioning: %w", err)
		}

		ns := customerID.Namespace

		accountIDs, err := s.Repo.GetCustomerAccountIDs(ctx, customerID)
		if err != nil {
			return ledger.CustomerAccounts{}, fmt.Errorf("failed to get customer account IDs: %w", err)
		}

		for _, accountType := range ledger.CustomerAccountTypes {
			if _, ok := accountIDs[accountType]; ok {
				continue
			}

			acc, err := s.AccountService.CreateAccount(ctx, ledgeraccount.CreateAccountInput{
				Namespace: ns,
				Type:      accountType,
			})
			if err != nil {
				return ledger.CustomerAccounts{}, fmt.Errorf("failed to create %s account: %w", accountType, err)
			}

			if err := s.Repo.CreateCustomerAccount(ctx, CreateCustomerAccountInput{
				CustomerID:  customerID,
				AccountType: accountType,
				AccountID:   acc.ID().ID,
			}); err != nil {
				if existingErr, ok := AsCustomerAccountAlreadyExistsError(err); ok {
					// Idempotent create semantics: if mapping already exists, use it.
					accountIDs[accountType] = existingErr.AccountID
					continue
				}

				return ledger.CustomerAccounts{}, fmt.Errorf("failed to create %s account mapping: %w", accountType, err)
			}
		}

		return s.GetCustomerAccounts(ctx, customerID)
	})
}

// GetCustomerAccounts retrieves the FBO and Receivable accounts for a customer.
func (s *AccountResolver) GetCustomerAccounts(ctx context.Context, customerID customer.CustomerID) (ledger.CustomerAccounts, error) {
	ns := customerID.Namespace

	accountIDs, err := s.Repo.GetCustomerAccountIDs(ctx, customerID)
	if err != nil {
		return ledger.CustomerAccounts{}, fmt.Errorf("failed to get customer account IDs: %w", err)
	}

	fboID, ok := accountIDs[ledger.AccountTypeCustomerFBO]
	if !ok {
		return ledger.CustomerAccounts{}, ledger.ErrCustomerAccountMissing.WithAttrs(models.Attributes{
			"namespace":    customerID.Namespace,
			"customer_id":  customerID.ID,
			"account_type": ledger.AccountTypeCustomerFBO,
		})
	}

	receivableID, ok := accountIDs[ledger.AccountTypeCustomerReceivable]
	if !ok {
		return ledger.CustomerAccounts{}, ledger.ErrCustomerAccountMissing.WithAttrs(models.Attributes{
			"namespace":    customerID.Namespace,
			"customer_id":  customerID.ID,
			"account_type": ledger.AccountTypeCustomerReceivable,
		})
	}

	accruedID, ok := accountIDs[ledger.AccountTypeCustomerAccrued]
	if !ok {
		return ledger.CustomerAccounts{}, ledger.ErrCustomerAccountMissing.WithAttrs(models.Attributes{
			"namespace":    customerID.Namespace,
			"customer_id":  customerID.ID,
			"account_type": ledger.AccountTypeCustomerAccrued,
		})
	}

	fboAcc, err := s.AccountService.GetAccountByID(ctx, models.NamespacedID{Namespace: ns, ID: fboID})
	if err != nil {
		return ledger.CustomerAccounts{}, fmt.Errorf("failed to get FBO account: %w", err)
	}

	fboAccount, err := fboAcc.AsCustomerFBOAccount()
	if err != nil {
		return ledger.CustomerAccounts{}, err
	}

	receivableAcc, err := s.AccountService.GetAccountByID(ctx, models.NamespacedID{Namespace: ns, ID: receivableID})
	if err != nil {
		return ledger.CustomerAccounts{}, fmt.Errorf("failed to get Receivable account: %w", err)
	}

	receivableAccount, err := receivableAcc.AsCustomerReceivableAccount()
	if err != nil {
		return ledger.CustomerAccounts{}, err
	}

	accruedAcc, err := s.AccountService.GetAccountByID(ctx, models.NamespacedID{Namespace: ns, ID: accruedID})
	if err != nil {
		return ledger.CustomerAccounts{}, fmt.Errorf("failed to get Accrued account: %w", err)
	}

	accruedAccount, err := accruedAcc.AsCustomerAccruedAccount()
	if err != nil {
		return ledger.CustomerAccounts{}, err
	}

	return ledger.CustomerAccounts{
		FBOAccount:        fboAccount,
		ReceivableAccount: receivableAccount,
		AccruedAccount:    accruedAccount,
	}, nil
}

func (s *AccountResolver) EnsureBusinessAccounts(ctx context.Context, namespace string) (ledger.BusinessAccounts, error) {
	return transaction.Run(ctx, s.Repo, func(ctx context.Context) (ledger.BusinessAccounts, error) {
		if err := s.lockBusinessProvisioning(ctx, namespace); err != nil {
			return ledger.BusinessAccounts{}, fmt.Errorf("failed to lock business account provisioning: %w", err)
		}

		existing, err := s.listBusinessAccountsByType(ctx, namespace)
		if err != nil {
			return ledger.BusinessAccounts{}, err
		}

		for _, accountType := range ledger.BusinessAccountTypes {
			if _, ok := existing[accountType]; ok {
				continue
			}

			acc, err := s.AccountService.CreateAccount(ctx, ledgeraccount.CreateAccountInput{
				Namespace: namespace,
				Type:      accountType,
			})
			if err != nil {
				return ledger.BusinessAccounts{}, fmt.Errorf("failed to create %s account: %w", accountType, err)
			}

			existing[accountType] = acc
		}

		return s.GetBusinessAccounts(ctx, namespace)
	})
}

// GetBusinessAccounts retrieves the shared business accounts for a namespace.
func (s *AccountResolver) GetBusinessAccounts(ctx context.Context, namespace string) (ledger.BusinessAccounts, error) {
	existing, err := s.listBusinessAccountsByType(ctx, namespace)
	if err != nil {
		return ledger.BusinessAccounts{}, err
	}

	wash, ok := existing[ledger.AccountTypeWash]
	if !ok {
		return ledger.BusinessAccounts{}, ledger.ErrBusinessAccountMissing.WithAttrs(models.Attributes{
			"namespace":    namespace,
			"account_type": ledger.AccountTypeWash,
		})
	}

	earnings, ok := existing[ledger.AccountTypeEarnings]
	if !ok {
		return ledger.BusinessAccounts{}, ledger.ErrBusinessAccountMissing.WithAttrs(models.Attributes{
			"namespace":    namespace,
			"account_type": ledger.AccountTypeEarnings,
		})
	}

	brokerage, ok := existing[ledger.AccountTypeBrokerage]
	if !ok {
		return ledger.BusinessAccounts{}, ledger.ErrBusinessAccountMissing.WithAttrs(models.Attributes{
			"namespace":    namespace,
			"account_type": ledger.AccountTypeBrokerage,
		})
	}

	washAcc, err := wash.AsBusinessAccount()
	if err != nil {
		return ledger.BusinessAccounts{}, err
	}

	earningsAcc, err := earnings.AsBusinessAccount()
	if err != nil {
		return ledger.BusinessAccounts{}, err
	}

	brokerageAcc, err := brokerage.AsBusinessAccount()
	if err != nil {
		return ledger.BusinessAccounts{}, err
	}

	return ledger.BusinessAccounts{
		WashAccount:      washAcc,
		EarningsAccount:  earningsAcc,
		BrokerageAccount: brokerageAcc,
	}, nil
}

func (s *AccountResolver) listBusinessAccountsByType(ctx context.Context, namespace string) (map[ledger.AccountType]*ledgeraccount.Account, error) {
	existing, err := s.AccountService.ListAccounts(ctx, ledgeraccount.ListAccountsInput{
		Namespace:    namespace,
		AccountTypes: ledger.BusinessAccountTypes,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list business accounts: %w", err)
	}

	byType := make(map[ledger.AccountType]*ledgeraccount.Account, len(existing))
	for _, acc := range existing {
		if prev, ok := byType[acc.Type()]; ok {
			return nil, fmt.Errorf("multiple %s accounts found in namespace %s: %s and %s", acc.Type(), namespace, prev.ID().ID, acc.ID().ID)
		}

		byType[acc.Type()] = acc
	}

	return byType, nil
}

func (s *AccountResolver) lockCustomerProvisioning(ctx context.Context, customerID customer.CustomerID) error {
	if s.Locker == nil {
		return nil
	}

	key, err := lockr.NewKey(
		"ledger",
		"customer_accounts",
		"namespace",
		customerID.Namespace,
		"customer",
		customerID.ID,
	)
	if err != nil {
		return err
	}

	return s.Locker.LockForTX(ctx, key)
}

func (s *AccountResolver) lockBusinessProvisioning(ctx context.Context, namespace string) error {
	if s.Locker == nil {
		return nil
	}

	key, err := lockr.NewKey(
		"ledger",
		"business_accounts",
		"namespace",
		namespace,
	)
	if err != nil {
		return err
	}

	return s.Locker.LockForTX(ctx, key)
}
