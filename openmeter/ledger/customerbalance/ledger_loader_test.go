package customerbalance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/creditvoid"
	ledgerhistorical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestLedgerCreditTransactionLoaderDoesNotRetainHasMoreAfterHiddenFinalPage(t *testing.T) {
	now := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	visible := ledgerLoaderTestTransaction(t, "tx-visible", now.Add(-time.Minute), nil, ledger.AccountTypeCustomerAccrued)
	hidden := ledgerLoaderTestTransaction(t, "tx-hidden", now.Add(-2*time.Minute), models.Annotations{
		creditvoid.AnnotationCreditVoidRecordID: "void-record-1",
	}, ledger.AccountTypeCustomerReceivable)

	next := visible.Cursor()
	fakeLedger := &ledgerLoaderFakeLedger{
		pages: []ledger.ListTransactionsResult{
			{
				Items:      []ledger.Transaction{visible},
				NextCursor: &next,
			},
			{
				Items: []ledger.Transaction{hidden},
			},
		},
	}

	loader := newLedgerCreditTransactionLoader(
		&service{Ledger: fakeLedger},
		ledger.ListTransactionsCreditMovementNegative,
	)

	currency := currencyx.Code("USD")
	got, err := loader.Load(t.Context(), creditTransactionLoaderInput{
		Limit:      1,
		CustomerID: customer.CustomerID{Namespace: "ns", ID: "customer-id"},
		AccountID:  "fbo-account",
		Currency:   &currency,
		AsOf:       now,
	})
	require.NoError(t, err)
	require.False(t, got.HasMore)
	require.Len(t, got.Items, 1)
	require.Equal(t, "tx-visible", got.Items[0].ID.ID)
	require.Len(t, fakeLedger.inputs, 2)
	require.NotNil(t, fakeLedger.inputs[1].Cursor)
}

func TestLedgerCreditTransactionLoaderScansPastHiddenRowsUntilLimit(t *testing.T) {
	now := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	hidden := ledgerLoaderTestTransaction(t, "tx-hidden", now.Add(-time.Minute), models.Annotations{
		creditvoid.AnnotationCreditVoidRecordID: "void-record-1",
	}, ledger.AccountTypeCustomerReceivable)
	visibleA := ledgerLoaderTestTransaction(t, "tx-visible-a", now.Add(-2*time.Minute), nil, ledger.AccountTypeCustomerAccrued)
	visibleB := ledgerLoaderTestTransaction(t, "tx-visible-b", now.Add(-3*time.Minute), nil, ledger.AccountTypeCustomerAccrued)

	next := hidden.Cursor()
	fakeLedger := &ledgerLoaderFakeLedger{
		pages: []ledger.ListTransactionsResult{
			{
				Items:      []ledger.Transaction{hidden},
				NextCursor: &next,
			},
			{
				Items: []ledger.Transaction{visibleA, visibleB},
			},
		},
	}

	loader := newLedgerCreditTransactionLoader(
		&service{Ledger: fakeLedger},
		ledger.ListTransactionsCreditMovementNegative,
	)

	currency := currencyx.Code("USD")
	got, err := loader.Load(t.Context(), creditTransactionLoaderInput{
		Limit:      2,
		CustomerID: customer.CustomerID{Namespace: "ns", ID: "customer-id"},
		AccountID:  "fbo-account",
		Currency:   &currency,
		AsOf:       now,
	})
	require.NoError(t, err)
	require.False(t, got.HasMore)
	require.Len(t, got.Items, 2)
	require.Equal(t, "tx-visible-a", got.Items[0].ID.ID)
	require.Equal(t, "tx-visible-b", got.Items[1].ID.ID)
	require.Len(t, fakeLedger.inputs, 2)
	require.Equal(t, 3, fakeLedger.inputs[0].Limit)
	require.NotNil(t, fakeLedger.inputs[1].Cursor)
}

type ledgerLoaderFakeLedger struct {
	pages  []ledger.ListTransactionsResult
	inputs []ledger.ListTransactionsInput
}

func (l *ledgerLoaderFakeLedger) CommitGroup(context.Context, ledger.TransactionGroupInput) (ledger.TransactionGroup, error) {
	return nil, errors.New("unexpected CommitGroup call")
}

func (l *ledgerLoaderFakeLedger) GetTransactionGroup(context.Context, models.NamespacedID) (ledger.TransactionGroup, error) {
	return nil, errors.New("unexpected GetTransactionGroup call")
}

func (l *ledgerLoaderFakeLedger) ListTransactions(_ context.Context, input ledger.ListTransactionsInput) (ledger.ListTransactionsResult, error) {
	l.inputs = append(l.inputs, input)
	if len(l.pages) == 0 {
		return ledger.ListTransactionsResult{}, errors.New("unexpected ListTransactions call")
	}

	page := l.pages[0]
	l.pages = l.pages[1:]

	return page, nil
}

func ledgerLoaderTestTransaction(
	t *testing.T,
	id string,
	bookedAt time.Time,
	annotations models.Annotations,
	offsetAccountType ledger.AccountType,
) ledger.Transaction {
	t.Helper()

	route := ledger.Route{Currency: currencyx.Code("USD")}
	key, err := ledger.BuildRoutingKey(route)
	require.NoError(t, err)

	tx, err := ledgerhistorical.NewTransactionFromData(ledgerhistorical.TransactionData{
		ID:          id,
		Namespace:   "ns",
		Annotations: annotations,
		CreatedAt:   bookedAt,
		BookedAt:    bookedAt,
	}, []ledgerhistorical.EntryData{
		{
			ID:            id + "-fbo",
			Namespace:     "ns",
			CreatedAt:     bookedAt,
			SubAccountID:  "fbo-subaccount",
			AccountType:   ledger.AccountTypeCustomerFBO,
			Route:         route,
			RouteID:       "fbo-route",
			RouteKey:      key.Value(),
			RouteKeyVer:   key.Version(),
			Amount:        alpacadecimal.NewFromInt(-10),
			TransactionID: id,
		},
		{
			ID:            id + "-offset",
			Namespace:     "ns",
			CreatedAt:     bookedAt,
			SubAccountID:  "offset-subaccount",
			AccountType:   offsetAccountType,
			Route:         route,
			RouteID:       "offset-route",
			RouteKey:      key.Value(),
			RouteKeyVer:   key.Version(),
			Amount:        alpacadecimal.NewFromInt(10),
			TransactionID: id,
		},
	})
	require.NoError(t, err)

	return tx
}
