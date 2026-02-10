package historical

import (
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CreateEntryInput struct {
	Namespace string
	// Annotations models.Annotations // TBD

	AccountID    string
	DimensionIDs []string

	Amount        alpacadecimal.Decimal
	TransactionID string
}

type EntryInput struct {
	input   CreateEntryInput
	address ledger.Address
}

// ----------------------------------------------------------------------------
// Let's implement ledger.EntryInput interface
// ----------------------------------------------------------------------------

var _ ledger.EntryInput = (*EntryInput)(nil)

func (e *EntryInput) Account() ledger.Address {
	return e.address
}

func (e *EntryInput) Amount() alpacadecimal.Decimal {
	return e.input.Amount
}

type EntryData struct {
	ID          string
	Namespace   string
	Annotations models.Annotations
	CreatedAt   time.Time

	AccountID          string
	AccountType        ledger.AccountType
	DimensionIDs       []string
	DimensionsExpanded map[string]*account.Dimension

	Amount        alpacadecimal.Decimal
	TransactionID string
}

type Entry struct {
	data EntryData
}

var _ ledger.Entry = (*Entry)(nil)

// ----------------------------------------------------------------------------
// Let's implement ledger.Entry interface
// ----------------------------------------------------------------------------

func (e *Entry) Account() ledger.Address {
	return account.NewAddressFromData(account.AddressData{
		ID:          models.NamespacedID{Namespace: e.data.Namespace, ID: e.data.AccountID},
		AccountType: e.data.AccountType,
		Dimensions:  e.data.DimensionsExpanded,
	})
}

func (e *Entry) Amount() alpacadecimal.Decimal {
	return e.data.Amount
}

func (e *Entry) TransactionID() models.NamespacedID {
	return models.NamespacedID{Namespace: e.data.Namespace, ID: e.data.TransactionID}
}
