package ledger

import (
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

type AccountType string

// Customer Accounts
const (
	AccountTypeCustomerFBO        AccountType = "customer_fbo" // is this the right name?
	AccountTypeCustomerReceivable AccountType = "customer_receivable"
	AccountTypeCustomerBreakage   AccountType = "customer_breakage"
)

type CustomerAccounts struct {
	FBOAccountID        string
	ReceivableAccountID string
	BreakageAccountID   string
}

// Shared Business Accounts
const (
	AccountTypeWash      AccountType = "wash"
	AccountTypeEarnings  AccountType = "earnings"
	AccountTypeBrokerage AccountType = "brokerage"
)

type BusinessAccounts struct {
	WashAccountID      string
	EarningsAccountID  string
	BrokerageAccountID string
}

func (t AccountType) Validate() error {
	switch t {
	case AccountTypeCustomerFBO, AccountTypeCustomerReceivable, AccountTypeCustomerBreakage:
		return nil
	case AccountTypeWash, AccountTypeEarnings, AccountTypeBrokerage:
		return nil
	default:
		return models.NewGenericValidationError(fmt.Errorf("invalid account type: %s", t))
	}
}
