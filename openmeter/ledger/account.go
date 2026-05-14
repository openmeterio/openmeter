package ledger

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
)

type AccountType string

// Customer Accounts
const (
	AccountTypeCustomerFBO        AccountType = "customer_fbo" // is this the right name?
	AccountTypeCustomerReceivable AccountType = "customer_receivable"
	AccountTypeCustomerAccrued    AccountType = "customer_accrued"
	// AccountTypeCustomerBreakage   AccountType = "customer_breakage"
)

type CustomerAccounts struct {
	FBOAccount        CustomerFBOAccount
	ReceivableAccount CustomerReceivableAccount
	AccruedAccount    CustomerAccruedAccount
	// BreakageAccount   Account
}

// Shared Business Accounts
const (
	AccountTypeWash      AccountType = "wash"
	AccountTypeEarnings  AccountType = "earnings"
	AccountTypeBrokerage AccountType = "brokerage"
)

var CustomerAccountTypes = []AccountType{
	AccountTypeCustomerFBO,
	AccountTypeCustomerReceivable,
	AccountTypeCustomerAccrued,
}

var BusinessAccountTypes = []AccountType{
	AccountTypeWash,
	AccountTypeEarnings,
	AccountTypeBrokerage,
}

type BusinessAccounts struct {
	WashAccount      BusinessAccount
	EarningsAccount  BusinessAccount
	BrokerageAccount BusinessAccount
}

func (t AccountType) Validate() error {
	switch t {
	case AccountTypeCustomerFBO, AccountTypeCustomerReceivable, AccountTypeCustomerAccrued:
		return nil
	case AccountTypeWash, AccountTypeEarnings, AccountTypeBrokerage:
		return nil
	default:
		return models.NewGenericValidationError(fmt.Errorf("invalid account type: %s", t))
	}
}

type AccountResolver interface {
	GetCustomerAccounts(ctx context.Context, customerID customer.CustomerID) (CustomerAccounts, error)
	EnsureBusinessAccounts(ctx context.Context, namespace string) (BusinessAccounts, error)
	GetBusinessAccounts(ctx context.Context, namespace string) (BusinessAccounts, error)
}

type AccountReader interface {
	GetAccountByID(ctx context.Context, id models.NamespacedID) (Account, error)
	GetSubAccountByID(ctx context.Context, id models.NamespacedID) (SubAccount, error)

	ListSubAccounts(ctx context.Context, input ListSubAccountsInput) ([]SubAccount, error)
	ListAccounts(ctx context.Context, input ListAccountsInput) ([]Account, error)
}

type AccountProvisioner interface {
	CreateAccount(ctx context.Context, input CreateAccountInput) (Account, error)
	EnsureSubAccount(ctx context.Context, input CreateSubAccountInput) (SubAccount, error)
}

type AccountCatalog interface {
	AccountReader
	AccountProvisioner
}

type AccountLocker interface {
	LockAccountsForPosting(ctx context.Context, accounts []Account) error
}

type ListAccountsInput struct {
	Namespace    string
	AccountTypes []AccountType
}

type ListSubAccountsInput struct {
	Namespace string
	AccountID string

	Route RouteFilter
}

type CreateAccountInput struct {
	Namespace   string
	Type        AccountType
	Annotations models.Annotations
}

func (c CreateAccountInput) Validate() error {
	if err := c.Type.Validate(); err != nil {
		return err
	}

	return nil
}

type CreateSubAccountInput struct {
	Namespace   string
	AccountID   string
	Annotations models.Annotations
	Route       Route
}

func (c CreateSubAccountInput) Validate() error {
	if c.AccountID == "" {
		return models.NewGenericValidationError(fmt.Errorf("account id is required"))
	}

	if c.Namespace == "" {
		return models.NewGenericValidationError(fmt.Errorf("namespace is required"))
	}

	if err := c.Route.Validate(); err != nil {
		return models.NewGenericValidationError(err)
	}

	return nil
}
