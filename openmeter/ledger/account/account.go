package account

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

// SubAccountCreatorLister is used by account types to find-or-create sub-accounts
// for a given route.
type SubAccountCreatorLister interface {
	ListSubAccounts(ctx context.Context, input ledger.ListSubAccountsInput) ([]ledger.SubAccount, error)
	EnsureSubAccount(ctx context.Context, input ledger.CreateSubAccountInput) (ledger.SubAccount, error)
}

type AccountLiveServices struct {
	SubAccountService SubAccountCreatorLister
}

// AccountData is a simple data transfer object for the Account entity.
type AccountData struct {
	ID          models.NamespacedID
	Annotations models.Annotations
	models.ManagedModel
	AccountType ledger.AccountType
}

func NewAccountFromData(data AccountData, services AccountLiveServices) (ledger.Account, error) {
	base := &Account{
		data:     data,
		services: services,
	}

	switch data.AccountType {
	case ledger.AccountTypeCustomerFBO:
		return newCustomerFBOAccount(base), nil
	case ledger.AccountTypeCustomerReceivable:
		return newCustomerReceivableAccount(base), nil
	case ledger.AccountTypeCustomerAccrued:
		return newCustomerAccruedAccount(base), nil
	case ledger.AccountTypeWash, ledger.AccountTypeEarnings, ledger.AccountTypeBrokerage:
		return newBusinessAccount(base), nil
	default:
		if err := data.AccountType.Validate(); err != nil {
			return nil, err
		}

		return base, nil
	}
}

// Account instance represent a given account
type Account struct {
	data AccountData

	services AccountLiveServices
}

var _ ledger.Account = (*Account)(nil)

func (a *Account) Type() ledger.AccountType {
	return a.data.AccountType
}

// ID returns the namespaced identifier of this account.
func (a *Account) ID() models.NamespacedID {
	return a.data.ID
}
