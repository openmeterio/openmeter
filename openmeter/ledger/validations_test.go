package ledger_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestValidateTransactionInputEntryAmountPrecision(t *testing.T) {
	tests := []struct {
		name     string
		currency currencyx.Code
		amount   string
		wantErr  bool
	}{
		{
			name:     "USD accepts cents",
			currency: currencyx.Code("USD"),
			amount:   "10.01",
		},
		{
			name:     "USD rejects sub-cent amount",
			currency: currencyx.Code("USD"),
			amount:   "10.001",
			wantErr:  true,
		},
		{
			name:     "USD accepts negative cents",
			currency: currencyx.Code("USD"),
			amount:   "-10.01",
		},
		{
			name:     "JPY accepts whole amount",
			currency: currencyx.Code("JPY"),
			amount:   "10",
		},
		{
			name:     "JPY rejects fractional amount",
			currency: currencyx.Code("JPY"),
			amount:   "10.1",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount := mustDecimal(t, tt.amount)
			address := mustPostingAddress(t, tt.currency)
			txInput := &testutils.AnyTransactionInput{
				BookedAtValue: time.Now(),
				EntryInputsValues: []*testutils.AnyEntryInput{
					{
						Address:     address,
						AmountValue: amount,
					},
					{
						Address:     address,
						AmountValue: amount.Neg(),
					},
				},
			}

			err := ledger.ValidateTransactionInput(t.Context(), txInput)
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			require.ErrorIs(t, err, ledger.ErrTransactionAmountInvalid)

			issues, issueErr := models.AsValidationIssues(err)
			require.NoError(t, issueErr)
			require.Len(t, issues, 1)
			require.Equal(t, ledger.ErrCodeTransactionAmountInvalid, issues[0].Code())

			attrs := issues[0].Attributes()
			require.Equal(t, "amount_not_rounded_to_currency_precision", attrs["reason"])
			require.Equal(t, tt.currency, attrs["currency"])
			require.Equal(t, amount.String(), attrs["amount"])
			require.NotEmpty(t, attrs["rounded_amount"])
		})
	}
}

func TestListTransactionsInputValidateRouteFilter(t *testing.T) {
	costBasis := alpacadecimal.NewFromFloat(0.7)
	taxCode := "vat"
	taxBehavior := ledger.TaxBehaviorInclusive
	creditPriority := 1
	authStatus := ledger.TransactionAuthorizationStatusAuthorized

	tests := []struct {
		name    string
		route   ledger.RouteFilter
		wantErr bool
	}{
		{
			name: "currency route filter is supported",
			route: ledger.RouteFilter{
				Currency: currencyx.Code("USD"),
			},
		},
		{
			name: "exact features route filter is supported",
			route: ledger.RouteFilter{
				Features: mo.Some([]string{"feature-a"}),
			},
		},
		{
			name: "match feature route filter is supported",
			route: ledger.RouteFilter{
				MatchFeature: "feature-a",
			},
		},
		{
			name: "cost basis route filter is rejected",
			route: ledger.RouteFilter{
				CostBasis: mo.Some(&costBasis),
			},
			wantErr: true,
		},
		{
			name: "tax code route filter is rejected",
			route: ledger.RouteFilter{
				TaxCode: mo.Some(&taxCode),
			},
			wantErr: true,
		},
		{
			name: "tax behavior route filter is rejected",
			route: ledger.RouteFilter{
				TaxBehavior: mo.Some(&taxBehavior),
			},
			wantErr: true,
		},
		{
			name: "credit priority route filter is rejected",
			route: ledger.RouteFilter{
				CreditPriority: &creditPriority,
			},
			wantErr: true,
		},
		{
			name: "transaction authorization route filter is rejected",
			route: ledger.RouteFilter{
				TransactionAuthorizationStatus: &authStatus,
			},
			wantErr: true,
		},
		{
			name: "exact features and match feature cannot be combined",
			route: ledger.RouteFilter{
				Features:     mo.Some([]string{"feature-a"}),
				MatchFeature: "feature-a",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ledger.ListTransactionsInput{
				Namespace: "ns-test",
				Limit:     1,
				Route:     tt.route,
			}.Validate()

			if tt.wantErr {
				require.Error(t, err)
				require.ErrorIs(t, err, ledger.ErrListTransactionsInputInvalid)
				return
			}

			require.NoError(t, err)
		})
	}
}

func mustPostingAddress(t *testing.T, currency currencyx.Code) ledger.PostingAddress {
	t.Helper()

	route := ledger.Route{Currency: currency}
	key, err := ledger.BuildRoutingKey(route)
	require.NoError(t, err)

	address, err := ledgeraccount.NewAddressFromData(ledgeraccount.AddressData{
		SubAccountID: "sub_" + string(currency),
		AccountType:  ledger.AccountTypeCustomerFBO,
		Route:        route,
		RouteID:      "route_" + string(currency),
		RoutingKey:   key,
	})
	require.NoError(t, err)

	return address
}

func mustDecimal(t *testing.T, raw string) alpacadecimal.Decimal {
	t.Helper()

	value, err := alpacadecimal.NewFromString(raw)
	require.NoError(t, err)

	return value
}
