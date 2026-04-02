package account

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// Balance represents the balance of an account.
type Balance struct {
	settled alpacadecimal.Decimal
	pending alpacadecimal.Decimal
}

var _ ledger.Balance = (*Balance)(nil)

func (b *Balance) Settled() alpacadecimal.Decimal {
	return b.settled
}

func (b *Balance) Pending() alpacadecimal.Decimal {
	return b.pending
}

// SubAccountCreatorLister is used by account types to find-or-create sub-accounts
// for a given route.
type SubAccountCreatorLister interface {
	ListSubAccounts(ctx context.Context, input ListSubAccountsInput) ([]*SubAccount, error)
	EnsureSubAccount(ctx context.Context, input CreateSubAccountInput) (*SubAccount, error)
}

type AccountLiveServices struct {
	Querier           ledger.Querier
	Locker            *lockr.Locker
	SubAccountService SubAccountCreatorLister
}

// AccountData is a simple data transfer object for the Account entity.
type AccountData struct {
	ID          models.NamespacedID
	Annotations models.Annotations
	models.ManagedModel
	AccountType ledger.AccountType
}

func NewAccountFromData(data AccountData, services AccountLiveServices) (*Account, error) {
	return &Account{
		data:     data,
		services: services,
	}, nil
}

// Account instance represent a given account
type Account struct {
	data AccountData

	services AccountLiveServices
}

// ----------------------------------------------------------------------------
// Let's implement ledger.Account interface
// ----------------------------------------------------------------------------

var _ ledger.Account = (*Account)(nil)

func (a *Account) GetBalance(ctx context.Context, query ledger.RouteFilter) (ledger.Balance, error) {
	// TODO: this is a hack
	// package boundary between account and historical ledger is incorrect, dependency resolution is broken
	if a.services.Querier == nil {
		return nil, fmt.Errorf("account %s in namespace %s cannot query balances: querier is not configured", a.data.ID.ID, a.data.ID.Namespace)
	}

	// We can store the last cursor and balance, this will be added later
	lastClosingCursor := (*pagination.Cursor)(nil)
	periodSinceListClosing := (*timeutil.OpenPeriod)(nil)
	startingBalance := alpacadecimal.NewFromInt(0)

	ledgerQuery := ledger.Query{
		Namespace: a.data.ID.Namespace,
		Cursor:    lastClosingCursor,
		Filters: ledger.Filters{
			BookedAtPeriod: periodSinceListClosing,
			AccountID:      lo.ToPtr(a.data.ID.ID),
			Route:          query,
		},
	}

	res, err := a.services.Querier.SumEntries(ctx, ledgerQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to sum entries for query %+v: %w", query, err)
	}

	return &Balance{
		settled: res.SettledSum.Add(startingBalance),
		pending: res.PendingSum.Add(startingBalance),
	}, nil
}

func (a *Account) Type() ledger.AccountType {
	return a.data.AccountType
}

// ID returns the namespaced identifier of this account.
func (a *Account) ID() models.NamespacedID {
	return a.data.ID
}
