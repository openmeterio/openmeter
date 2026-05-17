package historical

import (
	"fmt"
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
	IdentityKey string

	SubAccountID string
	AccountType  ledger.AccountType
	Route        ledger.Route
	RouteID      string
	RouteKey     string
	RouteKeyVer  ledger.RoutingKeyVersion

	Amount        alpacadecimal.Decimal
	TransactionID string
}

type Entry struct {
	data           EntryData
	postingAddress ledger.PostingAddress
}

var _ ledger.Entry = (*Entry)(nil)

// ----------------------------------------------------------------------------
// Let's implement ledger.Entry interface
// ----------------------------------------------------------------------------

func newEntryFromData(data EntryData) (*Entry, error) {
	routingKey, err := ledger.NewRoutingKey(data.RouteKeyVer, data.RouteKey)
	if err != nil {
		return nil, fmt.Errorf("routing key: %w", err)
	}

	addr, err := account.NewAddressFromData(account.AddressData{
		SubAccountID: data.SubAccountID,
		AccountType:  data.AccountType,
		Route:        data.Route,
		RouteID:      data.RouteID,
		RoutingKey:   routingKey,
	})
	if err != nil {
		return nil, err
	}

	return &Entry{data: data, postingAddress: addr}, nil
}

func (e *Entry) PostingAddress() ledger.PostingAddress {
	return e.postingAddress
}

func (e *Entry) Amount() alpacadecimal.Decimal {
	return e.data.Amount
}

func (e *Entry) IdentityKey() string {
	return e.data.IdentityKey
}

func (e *Entry) Annotations() models.Annotations {
	return e.data.Annotations
}

func (e *Entry) ID() models.NamespacedID {
	return models.NamespacedID{Namespace: e.data.Namespace, ID: e.data.ID}
}

func (e *Entry) TransactionID() models.NamespacedID {
	return models.NamespacedID{Namespace: e.data.Namespace, ID: e.data.TransactionID}
}
