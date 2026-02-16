package account

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/ledger"
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

// AccountData is a simple data transfer object for the Account entity.
type AccountData struct {
	ID          models.NamespacedID
	Annotations models.Annotations
	models.ManagedModel
	AccountType ledger.AccountType
}

func NewAccountFromData(querier ledger.Querier, data AccountData) *Account {
	return &Account{
		data:    data,
		querier: querier,
	}
}

// Account instance represent a given account
type Account struct {
	data AccountData

	dimensions map[string]*Dimension

	querier ledger.Querier
}

// ----------------------------------------------------------------------------
// Let's implement ledger.Account interface
// ----------------------------------------------------------------------------

var _ ledger.Account = (*Account)(nil)

// Returns the address of the account
func (a *Account) Address() ledger.Address {
	return NewAddressFromData(a.AddressData())
}

// Creates a new sub-account of the current account defined by the given dimensions.
// The new account will have the dimensions of both the current account as well as the provided list.
func (a *Account) SubAccount(dimensions ...ledger.Dimension) (ledger.Account, error) {
	sub := a.new()

	dMap := make(map[string]*Dimension)
	for _, dimension := range a.dimensions {
		dMap[dimension.Key()] = dimension
	}

	for _, dimension := range dimensions {
		if _, ok := dMap[dimension.Key()]; ok {
			return nil, fmt.Errorf("dimension %s is present more than once, first %v second %v", dimension, dMap[dimension.Key()], dimension)
		}

		d, ok := dimension.(*Dimension)
		if !ok {
			return nil, fmt.Errorf("dimension %T is not a *Dimension", dimension)
		}

		dMap[d.Key()] = d
	}

	sub.dimensions = dMap

	return sub, nil
}

func (a *Account) GetBalance(ctx context.Context) (ledger.Balance, error) {
	// We can store the last cursor and balance, this will be added later
	lastClosingCursor := (*pagination.Cursor)(nil)
	periodSinceListClosing := (*timeutil.OpenPeriod)(nil)
	startingBalance := alpacadecimal.NewFromInt(0)

	query := ledger.Query{
		Namespace: a.data.ID.Namespace,
		Cursor:    lastClosingCursor,
		Filters: ledger.Filters{
			BookedAtPeriod: periodSinceListClosing,
			Account:        a.Address(),
		},
	}

	res, err := a.querier.SumEntries(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to sum entries for query %+v: %w", query, err)
	}

	return &Balance{
		settled: res.SettledSum.Add(startingBalance),
		pending: res.PendingSum.Add(startingBalance),
	}, nil
}

// ----------------------------------------------------------------------------
// Implementation specific methods
// ----------------------------------------------------------------------------

func (a *Account) new() *Account {
	return NewAccountFromData(a.querier, a.data)
}

func (a *Account) AddressData() AddressData {
	return AddressData{
		ID:          a.data.ID,
		AccountType: a.data.AccountType,
		Dimensions:  a.dimensions,
	}
}

// Returns the root account without dimensions
func (a *Account) RootAccount(ctx context.Context) (ledger.Account, error) {
	root := a.new()
	return root, nil
}
