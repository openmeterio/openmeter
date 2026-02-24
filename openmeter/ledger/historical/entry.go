package historical

import (
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/models"
)

type EntryData struct {
	ID          string
	Namespace   string
	Annotations models.Annotations
	CreatedAt   time.Time

	AccountID   string
	AccountType ledger.AccountType

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

func (e *Entry) PostingAddress() ledger.PostingAddress {
	return account.NewAddressFromData(account.AddressData{
		SubAccountID: e.data.AccountID,
		AccountType:  e.data.AccountType,
	})
}

func (e *Entry) Amount() alpacadecimal.Decimal {
	return e.data.Amount
}

func (e *Entry) TransactionID() models.NamespacedID {
	return models.NamespacedID{Namespace: e.data.Namespace, ID: e.data.TransactionID}
}
