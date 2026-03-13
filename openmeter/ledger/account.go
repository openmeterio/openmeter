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
	// AccountTypeCustomerBreakage   AccountType = "customer_breakage"
)

type CustomerAccounts struct {
	FBOAccount        CustomerFBOAccount
	ReceivableAccount CustomerReceivableAccount
	// BreakageAccount   Account
}

// Shared Business Accounts
const (
	AccountTypeWash      AccountType = "wash"
	AccountTypeEarnings  AccountType = "earnings"
	AccountTypeBrokerage AccountType = "brokerage"
)

type BusinessAccounts struct {
	WashAccount      BusinessAccount
	EarningsAccount  BusinessAccount
	BrokerageAccount BusinessAccount
}

func (t AccountType) Validate() error {
	switch t {
	case AccountTypeCustomerFBO, AccountTypeCustomerReceivable:
		return nil
	case AccountTypeWash, AccountTypeEarnings, AccountTypeBrokerage:
		return nil
	default:
		return models.NewGenericValidationError(fmt.Errorf("invalid account type: %s", t))
	}
}

type AccountResolver interface {
	GetCustomerAccounts(ctx context.Context, customerID customer.CustomerID) (CustomerAccounts, error)
	GetBusinessAccounts(ctx context.Context, namespace string) (BusinessAccounts, error)
}
