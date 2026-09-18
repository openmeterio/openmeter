package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgerhistorical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestBackfilledCreditReissueRoutePreservesCostBasisCurrency(t *testing.T) {
	// given:
	// - a custom-currency FBO route funded through a USD exchange
	// when:
	// - a correction resolves the route for reissuing backfilled credit
	// then:
	// - the reissued route keeps the original fiat source
	now := time.Now().UTC()
	costBasis := alpacadecimal.NewFromFloat(0.25)
	priority := 1
	costBasisCurrency := lo.ToPtr(currencyx.Code("USD"))
	currency, err := currencies.ParseCurrencyReference([]byte("custom|v1|ACME|custom-currency-id|2"))
	require.NoError(t, err)
	route := ledger.Route{
		Currency:          currency,
		CostBasisCurrency: costBasisCurrency,
		CostBasis:         &costBasis,
		CreditPriority:    &priority,
	}
	key, err := ledger.BuildRoutingKey(route)
	require.NoError(t, err)

	transaction, err := ledgerhistorical.NewTransactionFromData(
		ledgerhistorical.TransactionData{
			ID:        "tx-1",
			Namespace: "ns",
			CreatedAt: now,
			BookedAt:  now,
		},
		[]ledgerhistorical.EntryData{
			{
				ID:            "entry-1",
				Namespace:     "ns",
				CreatedAt:     now,
				SubAccountID:  "subaccount-1",
				AccountType:   ledger.AccountTypeCustomerFBO,
				Route:         route,
				RouteID:       "route-1",
				RouteKey:      key.Value(),
				RouteKeyVer:   key.Version(),
				Amount:        alpacadecimal.NewFromInt(10),
				TransactionID: "tx-1",
			},
		},
	)
	require.NoError(t, err)

	group := ledgerhistorical.NewTransactionGroupFromData(
		ledgerhistorical.TransactionGroupData{
			ID:        "group-1",
			Namespace: "ns",
			CreatedAt: now,
		},
		[]*ledgerhistorical.Transaction{transaction},
	)

	resolved, err := (&service{}).backfilledCreditReissueRoute(group)
	require.NoError(t, err)
	require.Equal(t, costBasisCurrency, resolved.costBasisCurrency)
}
