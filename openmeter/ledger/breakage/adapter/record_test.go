package adapter

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	transactionstestutils "github.com/openmeterio/openmeter/openmeter/ledger/transactions/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestAdapter_ListExpiredRecordsFiltersByRoute(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "ledger-breakage-adapter")

	a, err := New(Config{Client: env.DB})
	require.NoError(t, err)

	expiresAt := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	asOf := expiresAt.Add(time.Minute)

	records := []breakage.Record{
		newExpiredRecord(t, env, env.Currency, "unrestricted", nil, expiresAt),
		newExpiredRecord(t, env, env.Currency, "feature-a", []string{"feature-a"}, expiresAt),
		newExpiredRecord(t, env, env.Currency, "feature-a-b", []string{"feature-a", "feature-b"}, expiresAt),
		newExpiredRecord(t, env, env.Currency, "feature-b", []string{"feature-b"}, expiresAt),
		newExpiredRecord(t, env, currencyx.Code("EUR"), "eur-unrestricted", nil, expiresAt),
	}
	require.NoError(t, a.CreateRecords(t.Context(), breakage.CreateRecordsInput{Records: records}))

	tests := []struct {
		name  string
		route ledger.RouteFilter
		want  []string
	}{
		{
			name: "unrestricted exact route",
			route: ledger.RouteFilter{
				Currency: env.Currency,
				Features: mo.Some[[]string](nil),
			},
			want: []string{"unrestricted"},
		},
		{
			name: "feature match route includes unrestricted and containing features",
			route: ledger.RouteFilter{
				Currency:     env.Currency,
				MatchFeature: "feature-a",
			},
			want: []string{"unrestricted", "feature-a", "feature-a-b"},
		},
		{
			name: "exact feature route",
			route: ledger.RouteFilter{
				Currency: env.Currency,
				Features: mo.Some([]string{"feature-b"}),
			},
			want: []string{"feature-b"},
		},
		{
			name: "currency route",
			route: ledger.RouteFilter{
				Currency: currencyx.Code("EUR"),
			},
			want: []string{"eur-unrestricted"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := a.ListExpiredRecords(t.Context(), breakage.ListExpiredRecordsInput{
				CustomerID: env.CustomerID,
				AsOf:       asOf,
				Route:      tt.route,
			})
			require.NoError(t, err)
			require.ElementsMatch(t, tt.want, recordNames(got))
		})
	}

	t.Run("feature match route with input currency", func(t *testing.T) {
		got, err := a.ListExpiredRecords(t.Context(), breakage.ListExpiredRecordsInput{
			CustomerID: env.CustomerID,
			Currency:   &env.Currency,
			AsOf:       asOf,
			Route: ledger.RouteFilter{
				MatchFeature: "feature-a",
			},
		})
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"unrestricted", "feature-a", "feature-a-b"}, recordNames(got))
	})
}

func newExpiredRecord(
	t *testing.T,
	env *ledgertestutils.IntegrationEnv,
	currency currencyx.Code,
	name string,
	features []string,
	expiresAt time.Time,
) breakage.Record {
	t.Helper()

	fboSubAccount, err := env.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:       currency,
		CreditPriority: ledger.DefaultCustomerFBOPriority,
		Features:       features,
	})
	require.NoError(t, err)

	breakageSubAccount, err := env.BusinessAccounts.BreakageAccount.GetSubAccountForRoute(t.Context(), ledger.BusinessRouteParams{
		Currency: currency,
	})
	require.NoError(t, err)

	txInput := &transactionstestutils.AnyTransactionInput{
		BookedAtValue: expiresAt,
		EntryInputsValues: []*transactionstestutils.AnyEntryInput{
			{
				Address:     fboSubAccount.Address(),
				AmountValue: alpacadecimal.NewFromInt(-1),
			},
			{
				Address:     breakageSubAccount.Address(),
				AmountValue: alpacadecimal.NewFromInt(1),
			},
		},
	}
	group, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), txInput.AsGroupInput(env.Namespace, nil))
	require.NoError(t, err)
	require.Len(t, group.Transactions(), 1)

	return breakage.Record{
		ID: models.NamespacedID{
			Namespace: env.CustomerID.Namespace,
			ID:        ulid.Make().String(),
		},
		Kind:                       ledger.BreakageKindPlan,
		Amount:                     alpacadecimal.NewFromInt(1),
		CustomerID:                 env.CustomerID,
		Currency:                   currency,
		CreditPriority:             ledger.DefaultCustomerFBOPriority,
		ExpiresAt:                  expiresAt,
		SourceKind:                 breakage.SourceKindCreditPurchase,
		BreakageTransactionGroupID: group.ID().ID,
		BreakageTransactionID:      group.Transactions()[0].ID().ID,
		FBOSubAccountID:            fboSubAccount.Address().SubAccountID(),
		BreakageSubAccountID:       breakageSubAccount.Address().SubAccountID(),
		Annotations: models.Annotations{
			"name": name,
		},
	}
}

func recordNames(records []breakage.Record) []string {
	names := make([]string, 0, len(records))
	for _, record := range records {
		name, _ := record.Annotations["name"].(string)
		names = append(names, name)
	}

	return names
}
