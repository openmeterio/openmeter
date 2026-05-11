package ledger_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
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

func mustPostingAddress(t *testing.T, currency currencyx.Code) ledger.PostingAddress {
	t.Helper()

	route := ledger.Route{Currency: currency}
	key, err := ledger.BuildRoutingKey(ledger.RoutingKeyVersionV1, route)
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
