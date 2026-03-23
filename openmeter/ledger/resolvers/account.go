package resolvers

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

// AccountResolver implements ledger.AccountResolver and manages customer ledger account provisioning.
type AccountResolver struct {
	AccountService ledgeraccount.Service
	Repo           CustomerAccountRepo
}

// AccountResolverConfig holds the dependencies for constructing a Service.
type AccountResolverConfig struct {
	AccountService ledgeraccount.Service
	Repo           CustomerAccountRepo
}

// NewAccountResolver constructs a resolvers Service from the given config.
func NewAccountResolver(cfg AccountResolverConfig) *AccountResolver {
	return &AccountResolver{
		AccountService: cfg.AccountService,
		Repo:           cfg.Repo,
	}
}

var _ ledger.AccountResolver = (*AccountResolver)(nil)

// CreateCustomerAccounts creates FBO and Receivable ledger accounts for a new customer
// and stores the mappings in the linking table.
func (s *AccountResolver) CreateCustomerAccounts(ctx context.Context, customerID customer.CustomerID) (ledger.CustomerAccounts, error) {
	return transaction.Run(ctx, s.Repo, func(ctx context.Context) (ledger.CustomerAccounts, error) {
		ns := customerID.Namespace

		accountIDs, err := s.Repo.GetCustomerAccountIDs(ctx, customerID)
		if err != nil {
			return ledger.CustomerAccounts{}, fmt.Errorf("failed to get customer account IDs: %w", err)
		}

		requiredTypes := []ledger.AccountType{
			ledger.AccountTypeCustomerFBO,
			ledger.AccountTypeCustomerReceivable,
			ledger.AccountTypeCustomerAccrued,
		}

		for _, accountType := range requiredTypes {
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

// GetBusinessAccounts retrieves (and idempotently creates) the shared business accounts
// for a namespace.
func (s *AccountResolver) GetBusinessAccounts(ctx context.Context, namespace string) (ledger.BusinessAccounts, error) {
	types := []ledger.AccountType{
		ledger.AccountTypeWash,
		ledger.AccountTypeEarnings,
		ledger.AccountTypeBrokerage,
	}

	existing, err := s.AccountService.ListAccounts(ctx, ledgeraccount.ListAccountsInput{
		Namespace:    namespace,
		AccountTypes: types,
	})
	if err != nil {
		return ledger.BusinessAccounts{}, fmt.Errorf("failed to list business accounts: %w", err)
	}

	byType := make(map[ledger.AccountType]*ledgeraccount.Account, len(existing))
	for _, acc := range existing {
		byType[acc.Type()] = acc
	}

	// Idempotently create any missing accounts
	for _, t := range types {
		if _, ok := byType[t]; !ok {
			acc, err := s.AccountService.CreateAccount(ctx, ledgeraccount.CreateAccountInput{
				Namespace: namespace,
				Type:      t,
			})
			if err != nil {
				return ledger.BusinessAccounts{}, fmt.Errorf("failed to create %s account: %w", t, err)
			}
			byType[t] = acc
		}
	}

	washAcc, err := byType[ledger.AccountTypeWash].AsBusinessAccount()
	if err != nil {
		return ledger.BusinessAccounts{}, err
	}

	earningsAcc, err := byType[ledger.AccountTypeEarnings].AsBusinessAccount()
	if err != nil {
		return ledger.BusinessAccounts{}, err
	}

	brokerageAcc, err := byType[ledger.AccountTypeBrokerage].AsBusinessAccount()
	if err != nil {
		return ledger.BusinessAccounts{}, err
	}

	return ledger.BusinessAccounts{
		WashAccount:      washAcc,
		EarningsAccount:  earningsAcc,
		BrokerageAccount: brokerageAcc,
	}, nil
}
