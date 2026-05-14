package ledger_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestEntryMatchesImpactFilter(t *testing.T) {
	t.Parallel()

	taxCode := "tax-standard"
	otherTaxCode := "tax-reduced"
	priority := 10
	otherPriority := 20
	costBasis := alpacadecimal.NewFromInt(1)
	otherCostBasis := alpacadecimal.NewFromInt(2)
	authStatus := ledger.TransactionAuthorizationStatusOpen
	otherAuthStatus := ledger.TransactionAuthorizationStatusAuthorized

	entry := mustImpactTestEntry(t, ledger.AccountTypeCustomerFBO, ledger.Route{
		Currency:                       currencyx.Code("USD"),
		TaxCode:                        &taxCode,
		Features:                       []string{"feature-a", "feature-b"},
		CostBasis:                      &costBasis,
		CreditPriority:                 &priority,
		TransactionAuthorizationStatus: &authStatus,
	})

	tests := []struct {
		name   string
		filter ledger.ImpactFilter
		want   bool
	}{
		{
			name: "empty filter matches",
			want: true,
		},
		{
			name: "account type matches",
			filter: ledger.ImpactFilter{
				AccountType: ledger.AccountTypeCustomerFBO,
			},
			want: true,
		},
		{
			name: "account type mismatch",
			filter: ledger.ImpactFilter{
				AccountType: ledger.AccountTypeCustomerAccrued,
			},
		},
		{
			name: "currency matches",
			filter: ledger.ImpactFilter{
				Route: ledger.RouteFilter{Currency: currencyx.Code("USD")},
			},
			want: true,
		},
		{
			name: "currency mismatch",
			filter: ledger.ImpactFilter{
				Route: ledger.RouteFilter{Currency: currencyx.Code("EUR")},
			},
		},
		{
			name: "tax code matches",
			filter: ledger.ImpactFilter{
				Route: ledger.RouteFilter{TaxCode: &taxCode},
			},
			want: true,
		},
		{
			name: "tax code mismatch",
			filter: ledger.ImpactFilter{
				Route: ledger.RouteFilter{TaxCode: &otherTaxCode},
			},
		},
		{
			name: "features match regardless of filter order",
			filter: ledger.ImpactFilter{
				Route: ledger.RouteFilter{Features: []string{"feature-b", "feature-a"}},
			},
			want: true,
		},
		{
			name: "features mismatch",
			filter: ledger.ImpactFilter{
				Route: ledger.RouteFilter{Features: []string{"feature-c"}},
			},
		},
		{
			name: "cost basis absent filter ignores route cost basis",
			filter: ledger.ImpactFilter{
				Route: ledger.RouteFilter{},
			},
			want: true,
		},
		{
			name: "cost basis matches",
			filter: ledger.ImpactFilter{
				Route: ledger.RouteFilter{CostBasis: mo.Some(&costBasis)},
			},
			want: true,
		},
		{
			name: "cost basis mismatch",
			filter: ledger.ImpactFilter{
				Route: ledger.RouteFilter{CostBasis: mo.Some(&otherCostBasis)},
			},
		},
		{
			name: "nil cost basis filter rejects non-nil route cost basis",
			filter: ledger.ImpactFilter{
				Route: ledger.RouteFilter{CostBasis: mo.Some[*alpacadecimal.Decimal](nil)},
			},
		},
		{
			name: "credit priority matches",
			filter: ledger.ImpactFilter{
				Route: ledger.RouteFilter{CreditPriority: &priority},
			},
			want: true,
		},
		{
			name: "credit priority mismatch",
			filter: ledger.ImpactFilter{
				Route: ledger.RouteFilter{CreditPriority: &otherPriority},
			},
		},
		{
			name: "authorization status matches",
			filter: ledger.ImpactFilter{
				Route: ledger.RouteFilter{TransactionAuthorizationStatus: &authStatus},
			},
			want: true,
		},
		{
			name: "authorization status mismatch",
			filter: ledger.ImpactFilter{
				Route: ledger.RouteFilter{TransactionAuthorizationStatus: &otherAuthStatus},
			},
		},
		{
			name: "multiple fields match together",
			filter: ledger.ImpactFilter{
				AccountType: ledger.AccountTypeCustomerFBO,
				Route: ledger.RouteFilter{
					Currency:       currencyx.Code("USD"),
					TaxCode:        &taxCode,
					Features:       []string{"feature-b", "feature-a"},
					CostBasis:      mo.Some(&costBasis),
					CreditPriority: &priority,
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, ledger.EntryMatchesImpactFilter(entry, tt.filter))
		})
	}
}

func TestEntryMatchesImpactFilter_NilRouteFields(t *testing.T) {
	t.Parallel()

	taxCode := "tax-standard"
	priority := 10
	authStatus := ledger.TransactionAuthorizationStatusOpen

	entry := mustImpactTestEntry(t, ledger.AccountTypeCustomerReceivable, ledger.Route{
		Currency: currencyx.Code("USD"),
	})

	tests := []struct {
		name   string
		filter ledger.ImpactFilter
		want   bool
	}{
		{
			name: "tax code required but route has nil",
			filter: ledger.ImpactFilter{
				Route: ledger.RouteFilter{TaxCode: &taxCode},
			},
		},
		{
			name: "nil cost basis required and route has nil",
			filter: ledger.ImpactFilter{
				Route: ledger.RouteFilter{CostBasis: mo.Some[*alpacadecimal.Decimal](nil)},
			},
			want: true,
		},
		{
			name: "credit priority required but route has nil",
			filter: ledger.ImpactFilter{
				Route: ledger.RouteFilter{CreditPriority: &priority},
			},
		},
		{
			name: "authorization status required but route has nil",
			filter: ledger.ImpactFilter{
				Route: ledger.RouteFilter{TransactionAuthorizationStatus: &authStatus},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, ledger.EntryMatchesImpactFilter(entry, tt.filter))
		})
	}
}

func TestTransactionImpact(t *testing.T) {
	t.Parallel()

	priorityOne := 1
	priorityTwo := 2

	tx := impactTestTransaction{
		entries: []ledger.Entry{
			mustImpactTestEntry(t, ledger.AccountTypeCustomerFBO, ledger.Route{
				Currency:       currencyx.Code("USD"),
				CreditPriority: &priorityOne,
			}, alpacadecimal.NewFromInt(10)),
			mustImpactTestEntry(t, ledger.AccountTypeCustomerFBO, ledger.Route{
				Currency:       currencyx.Code("USD"),
				CreditPriority: &priorityTwo,
			}, alpacadecimal.NewFromInt(-3)),
			mustImpactTestEntry(t, ledger.AccountTypeCustomerFBO, ledger.Route{
				Currency: currencyx.Code("EUR"),
			}, alpacadecimal.NewFromInt(7)),
			mustImpactTestEntry(t, ledger.AccountTypeCustomerAccrued, ledger.Route{
				Currency: currencyx.Code("USD"),
			}, alpacadecimal.NewFromInt(20)),
		},
	}

	tests := []struct {
		name   string
		filter ledger.ImpactFilter
		want   alpacadecimal.Decimal
	}{
		{
			name: "empty filter sums all entries",
			want: alpacadecimal.NewFromInt(34),
		},
		{
			name: "account type filter sums matching account type",
			filter: ledger.ImpactFilter{
				AccountType: ledger.AccountTypeCustomerFBO,
			},
			want: alpacadecimal.NewFromInt(14),
		},
		{
			name: "account type and currency filter sum matching entries",
			filter: ledger.ImpactFilter{
				AccountType: ledger.AccountTypeCustomerFBO,
				Route: ledger.RouteFilter{
					Currency: currencyx.Code("USD"),
				},
			},
			want: alpacadecimal.NewFromInt(7),
		},
		{
			name: "route priority filter sum matching entries",
			filter: ledger.ImpactFilter{
				AccountType: ledger.AccountTypeCustomerFBO,
				Route: ledger.RouteFilter{
					CreditPriority: &priorityOne,
				},
			},
			want: alpacadecimal.NewFromInt(10),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.True(t, tt.want.Equal(ledger.TransactionImpact(tx, tt.filter)))
		})
	}
}

type impactTestTransaction struct {
	entries []ledger.Entry
}

func (t impactTestTransaction) Cursor() ledger.TransactionCursor {
	return ledger.TransactionCursor{
		BookedAt:  t.BookedAt(),
		CreatedAt: t.BookedAt(),
		ID:        t.ID(),
	}
}

func (t impactTestTransaction) BookedAt() time.Time {
	return time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
}

func (t impactTestTransaction) Entries() []ledger.Entry {
	return t.entries
}

func (t impactTestTransaction) ID() models.NamespacedID {
	return models.NamespacedID{Namespace: "ns", ID: "tx-id"}
}

func (t impactTestTransaction) Annotations() models.Annotations {
	return nil
}

var _ ledger.Transaction = impactTestTransaction{}

type impactTestEntry struct {
	id       models.NamespacedID
	txID     models.NamespacedID
	address  ledger.PostingAddress
	amount   alpacadecimal.Decimal
	identity string
	metadata models.Annotations
}

func (e impactTestEntry) ID() models.NamespacedID {
	return e.id
}

func (e impactTestEntry) TransactionID() models.NamespacedID {
	return e.txID
}

func (e impactTestEntry) PostingAddress() ledger.PostingAddress {
	return e.address
}

func (e impactTestEntry) Amount() alpacadecimal.Decimal {
	return e.amount
}

func (e impactTestEntry) IdentityKey() string {
	return e.identity
}

func (e impactTestEntry) Annotations() models.Annotations {
	return e.metadata
}

var _ ledger.Entry = impactTestEntry{}

type impactTestAddress struct {
	subAccountID string
	accountType  ledger.AccountType
	route        ledger.SubAccountRoute
}

func (a impactTestAddress) SubAccountID() string {
	return a.subAccountID
}

func (a impactTestAddress) AccountType() ledger.AccountType {
	return a.accountType
}

func (a impactTestAddress) Route() ledger.SubAccountRoute {
	return a.route
}

func (a impactTestAddress) Equal(other ledger.PostingAddress) bool {
	return a.SubAccountID() == other.SubAccountID() &&
		a.AccountType() == other.AccountType() &&
		a.Route().ID() == other.Route().ID()
}

var _ ledger.PostingAddress = impactTestAddress{}

func mustImpactTestEntry(t *testing.T, accountType ledger.AccountType, route ledger.Route, amount ...alpacadecimal.Decimal) ledger.Entry {
	t.Helper()

	normalizedRoute, err := route.Normalize()
	require.NoError(t, err)

	routingKey, err := ledger.BuildRoutingKey(ledger.RoutingKeyVersionV1, normalizedRoute)
	require.NoError(t, err)

	subAccountRoute, err := ledger.NewSubAccountRouteFromData("route-id", routingKey, normalizedRoute)
	require.NoError(t, err)

	entryAmount := alpacadecimal.NewFromInt(1)
	if len(amount) > 0 {
		entryAmount = amount[0]
	}

	return impactTestEntry{
		id:   models.NamespacedID{Namespace: "ns", ID: "entry-id"},
		txID: models.NamespacedID{Namespace: "ns", ID: "tx-id"},
		address: impactTestAddress{
			subAccountID: "sub-account-id",
			accountType:  accountType,
			route:        subAccountRoute,
		},
		amount: entryAmount,
	}
}
