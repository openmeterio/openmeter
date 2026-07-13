package breakage_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	breakageadapter "github.com/openmeterio/openmeter/openmeter/ledger/breakage/adapter"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestPlanIssuanceCustomCurrencyPreservesSource(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "ledger-breakage")

	adapter, err := breakageadapter.New(breakageadapter.Config{Client: env.DB})
	require.NoError(t, err)

	service, err := breakage.NewService(breakage.Config{
		Adapter: adapter,
		Dependencies: transactions.ResolverDependencies{
			AccountService: env.Deps.ResolversService,
			AccountCatalog: env.Deps.AccountService,
			BalanceQuerier: env.Deps.HistoricalLedger,
		},
	})
	require.NoError(t, err)

	currency := currencyx.Code("ACME")
	source := currencyx.Code("USD")
	costBasis := alpacadecimal.NewFromFloat(0.5)
	expiresAt := env.Now().Add(24 * time.Hour)

	inputs, pending, err := service.PlanIssuance(t.Context(), breakage.PlanIssuanceInput{
		CustomerID: env.CustomerID,
		Amount:     alpacadecimal.NewFromInt(10),
		Currency:   currency,
		Source:     &source,
		CostBasis:  &costBasis,
		ExpiresAt:  expiresAt,
	})
	require.NoError(t, err)
	require.Len(t, pending, 1)

	group, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(env.Namespace, nil, inputs...))
	require.NoError(t, err)
	require.NoError(t, service.PersistCommittedRecords(t.Context(), pending, group))

	plans, err := service.ListPlans(t.Context(), breakage.ListPlansInput{
		CustomerID: env.CustomerID,
		Currency:   currency,
		AsOf:       env.Now(),
	})
	require.NoError(t, err)
	require.Len(t, plans, 1)

	fboRoute := plans[0].FBOAddress.Route().Route()
	require.Equal(t, currency, fboRoute.Currency)
	require.NotNil(t, fboRoute.Source)
	require.Equal(t, source, *fboRoute.Source)

	breakageRoute := plans[0].BreakageAddress.Route().Route()
	require.Equal(t, currency, breakageRoute.Currency)
	require.NotNil(t, breakageRoute.Source)
	require.Equal(t, source, *breakageRoute.Source)
}
