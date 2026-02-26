package resolvers

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Service implements ledger.AccountResolver and manages customer ledger account provisioning.
type Service struct {
	AccountService ledgeraccount.Service
	Repo           Repo
}

// ServiceConfig holds the dependencies for constructing a Service.
type ServiceConfig struct {
	AccountService ledgeraccount.Service
	Repo           Repo
}

// NewService constructs a resolvers Service from the given config.
func NewService(cfg ServiceConfig) *Service {
	return &Service{
		AccountService: cfg.AccountService,
		Repo:           cfg.Repo,
	}
}

var _ ledger.AccountResolver = (*Service)(nil)

// CreateCustomerAccounts creates FBO and Receivable ledger accounts for a new customer
// and stores the mappings in the linking table.
func (s *Service) CreateCustomerAccounts(ctx context.Context, customerID customer.CustomerID) (ledger.CustomerAccounts, error) {
	ns := customerID.Namespace

	// Create FBO account
	fboAcc, err := s.AccountService.CreateAccount(ctx, ledgeraccount.CreateAccountInput{
		Namespace: ns,
		Type:      ledger.AccountTypeCustomerFBO,
	})
	if err != nil {
		return ledger.CustomerAccounts{}, fmt.Errorf("failed to create FBO account: %w", err)
	}

	if err := s.Repo.CreateCustomerAccount(ctx, CreateCustomerAccountInput{
		CustomerID:  customerID,
		AccountType: ledger.AccountTypeCustomerFBO,
		AccountID:   fboAcc.ID().ID,
	}); err != nil {
		return ledger.CustomerAccounts{}, fmt.Errorf("failed to store FBO account mapping: %w", err)
	}

	// Create Receivable account
	receivableAcc, err := s.AccountService.CreateAccount(ctx, ledgeraccount.CreateAccountInput{
		Namespace: ns,
		Type:      ledger.AccountTypeCustomerReceivable,
	})
	if err != nil {
		return ledger.CustomerAccounts{}, fmt.Errorf("failed to create Receivable account: %w", err)
	}

	if err := s.Repo.CreateCustomerAccount(ctx, CreateCustomerAccountInput{
		CustomerID:  customerID,
		AccountType: ledger.AccountTypeCustomerReceivable,
		AccountID:   receivableAcc.ID().ID,
	}); err != nil {
		return ledger.CustomerAccounts{}, fmt.Errorf("failed to store Receivable account mapping: %w", err)
	}

	fboAccount, err := fboAcc.AsCustomerFBOAccount()
	if err != nil {
		return ledger.CustomerAccounts{}, err
	}

	receivableAccount, err := receivableAcc.AsCustomerReceivableAccount()
	if err != nil {
		return ledger.CustomerAccounts{}, err
	}

	return ledger.CustomerAccounts{
		FBOAccount:        fboAccount,
		ReceivableAccount: receivableAccount,
	}, nil
}

// GetCustomerAccounts retrieves the FBO and Receivable accounts for a customer.
func (s *Service) GetCustomerAccounts(ctx context.Context, customerID customer.CustomerID) (ledger.CustomerAccounts, error) {
	ns := customerID.Namespace

	accountIDs, err := s.Repo.GetCustomerAccountIDs(ctx, customerID)
	if err != nil {
		return ledger.CustomerAccounts{}, fmt.Errorf("failed to get customer account IDs: %w", err)
	}

	fboID, ok := accountIDs[ledger.AccountTypeCustomerFBO]
	if !ok {
		return ledger.CustomerAccounts{}, fmt.Errorf("FBO account not found for customer %s", customerID.ID)
	}

	receivableID, ok := accountIDs[ledger.AccountTypeCustomerReceivable]
	if !ok {
		return ledger.CustomerAccounts{}, fmt.Errorf("receivable account not found for customer %s", customerID.ID)
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

	return ledger.CustomerAccounts{
		FBOAccount:        fboAccount,
		ReceivableAccount: receivableAccount,
	}, nil
}

// GetBusinessAccounts retrieves (and idempotently creates) the shared business accounts
// for a namespace.
func (s *Service) GetBusinessAccounts(ctx context.Context, namespace string) (ledger.BusinessAccounts, error) {
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
