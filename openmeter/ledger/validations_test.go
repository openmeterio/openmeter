package ledger_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestValidateTransactionInputCurrencyAccounting(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		currency currencies.CurrencyReference
		amount   string
		wantErr  bool
	}{
		{
			name:     "accepts fiat precision",
			currency: currencies.NewCurrencyReference("USD"),
			amount:   "10.01",
		},
		{
			name:     "rejects excess fiat precision",
			currency: currencies.NewCurrencyReference("USD"),
			amount:   "10.001",
			wantErr:  true,
		},
		{
			name:     "accepts custom precision matching the route's declared precision",
			currency: mustCustomCurrencyReference(t, "CREDITS", 3),
			amount:   "10.001",
		},
		{
			name:     "rejects custom precision exceeding the route's declared precision",
			currency: mustCustomCurrencyReference(t, "CREDITS", 2),
			amount:   "10.001",
			wantErr:  true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			amount := mustDecimal(t, testCase.amount)
			address := mustPostingAddressWithCurrencyReference(t, testCase.currency)
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
			if testCase.wantErr {
				require.ErrorIs(t, err, ledger.ErrTransactionAmountInvalid)

				return
			}

			require.NoError(t, err)
		})
	}

	t.Run("rejects a globally balanced transaction that is unbalanced by currency", func(t *testing.T) {
		amount := mustDecimal(t, "25")
		txInput := &testutils.AnyTransactionInput{
			BookedAtValue: time.Now(),
			EntryInputsValues: []*testutils.AnyEntryInput{
				{
					Address:     mustPostingAddress(t, currencyx.Code("USD")),
					AmountValue: amount,
				},
				{
					Address:     mustPostingAddressWithCurrencyReference(t, mustCustomCurrencyReference(t, "ACME", 2)),
					AmountValue: amount.Neg(),
				},
			},
		}

		err := ledger.ValidateTransactionInput(t.Context(), txInput)
		require.Error(t, err)
		require.ErrorIs(t, err, ledger.ErrInvalidTransactionTotal)
	})
}

func TestListTransactionsInputValidateRouteFilter(t *testing.T) {
	costBasis := alpacadecimal.NewFromFloat(0.7)
	costBasisCurrency := currencyx.Code("USD")
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
				Currency: currencies.NewCurrencyReference(currencyx.Code("USD")),
			},
		},
		{
			name: "exchange source route filter is supported",
			route: ledger.RouteFilter{
				CostBasisCurrency: mo.Some(&costBasisCurrency),
			},
		},
		{
			name: "source-less route filter is supported",
			route: ledger.RouteFilter{
				CostBasisCurrency: mo.Some[*currencyx.Code](nil),
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

func mustCustomCurrencyReference(t *testing.T, code currencyx.Code, precision int) currencies.CurrencyReference {
	t.Helper()

	reference, err := currencies.ParseCurrencyReference([]byte(fmt.Sprintf("custom|v1|%s|custom-currency-id|%d", code, precision)))
	require.NoError(t, err)

	return reference
}

func mustPostingAddress(t *testing.T, currency currencyx.Code) ledger.PostingAddress {
	t.Helper()

	return mustPostingAddressWithCurrencyReference(t, currencies.NewCurrencyReference(currency))
}

func mustPostingAddressWithCurrencyReference(t *testing.T, currency currencies.CurrencyReference) ledger.PostingAddress {
	t.Helper()

	route := ledger.Route{Currency: currency}
	key, err := ledger.BuildRoutingKey(route)
	require.NoError(t, err)

	address, err := ledgeraccount.NewAddressFromData(ledgeraccount.AddressData{
		SubAccountID: "sub_" + string(currency.Code),
		AccountType:  ledger.AccountTypeCustomerFBO,
		Route:        route,
		RouteID:      "route_" + string(currency.Code),
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
