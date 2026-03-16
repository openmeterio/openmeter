package ledgerv2

import (
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

type AccountType string

// Customer Accounts
const (
	AccountTypeCustomerFBO        AccountType = "customer_fbo" // is this the right name?
	AccountTypeCustomerReceivable AccountType = "customer_receivable"
	// AccountTypeCustomerBreakage   AccountType = "customer_breakage"
)

// Shared Business Accounts
const (
	AccountTypeWash      AccountType = "wash"
	AccountTypeEarnings  AccountType = "earnings"
	AccountTypeBrokerage AccountType = "brokerage"
)

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

type BusinessAccounts struct {
	WashAccount      OrganizationalAccount
	EarningsAccount  OrganizationalAccount
	BrokerageAccount OrganizationalAccount
}

type CustomerAccounts struct {
	FBOAccount        CustomerAccount
	ReceivableAccount CustomerAccount
	// BreakageAccount   Account
}

type CustomerSubAccountDimensions = SubAccountDimensions
